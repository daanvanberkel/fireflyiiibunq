package main

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type BunqClient struct {
	config              *Config
	httpClient          *http.Client
	installation        BunqInstallation
	keyChain            *Keychain
	bunqServerPublicKey *rsa.PublicKey
	session             *BunqSessionServer
	log                 *logrus.Logger
}

func NewBunqClient(config *Config, log *logrus.Logger) (*BunqClient, error) {
	keyChain, err := NewKeyChain(config.StorageLocation+config.BunqConfig.PrivateKeyFileName, config.StorageLocation+config.BunqConfig.PublicKeyFileName, 2048)
	if err != nil {
		return nil, err
	}

	return &BunqClient{
		config:     config,
		httpClient: &http.Client{},
		keyChain:   keyChain,
		log:        log,
	}, nil
}

func (c *BunqClient) LoadInstallation() error {
	// Try to load installation from storage
	installationPath := c.config.StorageLocation + c.config.BunqConfig.InstallationFileName
	if _, err := os.Stat(installationPath); err == nil {
		data, err := os.ReadFile(installationPath)
		if err != nil {
			return err
		}

		var installation BunqInstallation
		if err := json.Unmarshal(data, &installation); err != nil {
			return err
		}

		c.installation = installation

		if err := c.parseBunqServerPublicKey(); err != nil {
			return err
		}

		return nil
	}

	// Generate new installation
	response, err := c.doBunqRequest("POST", "/installation", BunqInstallationRequest{
		ClientPublicKey: string(c.keyChain.PublicKeyPem),
	})
	if err != nil {
		return err
	}

	var installationResponse BunqInstallationResponse
	if err := json.Unmarshal(response, &installationResponse); err != nil {
		return err
	}

	installation := BunqInstallation{}
	for _, item := range installationResponse.Response {
		if item.Id != nil {
			installation.Id = item.Id
		}

		if item.Token != nil {
			installation.Token = item.Token
		}

		if item.ServerPublicKey != nil {
			installation.ServerPublicKey = item.ServerPublicKey
		}
	}

	c.installation = installation
	installationJson, err := json.Marshal(c.installation)
	if err != nil {
		return err
	}

	if err := os.WriteFile(installationPath, installationJson, 0700); err != nil {
		return err
	}

	if err := c.parseBunqServerPublicKey(); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) LoadDeviceServer() error {
	deviceServerPath := c.config.StorageLocation + c.config.BunqConfig.DeviceServerFileName
	if _, err := os.Stat(deviceServerPath); err == nil {
		return nil
	}

	response, err := c.doBunqRequest("POST", "/device-server", BunqDeviceServerRequest{
		Description:  c.config.BunqConfig.UserAgent,
		Secret:       c.config.BunqConfig.ApiKey,
		PermittedIps: c.config.BunqConfig.PermittedIps,
	})
	if err != nil {
		return err
	}

	var deviceServerResponse BunqDeviceServerResponse
	if err := json.Unmarshal(response, &deviceServerResponse); err != nil {
		return err
	}

	deviceServer := BunqDeviceServer{}
	for _, item := range deviceServerResponse.Response {
		if item.Id != nil {
			deviceServer.Id = item.Id
		}
	}

	deviceServerJson, err := json.Marshal(deviceServer)
	if err != nil {
		return err
	}

	if err := os.WriteFile(deviceServerPath, deviceServerJson, 0700); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) GetMonetaryBankAccounts() ([]*BunqMonetaryAccountBank, error) {
	if err := c.ensureSessionIsStarted(); err != nil {
		return nil, err
	}

	response, err := c.doBunqRequest("GET", "/user/"+strconv.Itoa(c.session.UserPerson.Id)+"/monetary-account-bank", nil)
	if err != nil {
		return nil, err
	}

	var monetaryBankAccountResponse BunqMonetaryAccountBankResponse
	if err := json.Unmarshal(response, &monetaryBankAccountResponse); err != nil {
		return nil, err
	}

	result := make([]*BunqMonetaryAccountBank, len(monetaryBankAccountResponse.Response))
	for i, account := range monetaryBankAccountResponse.Response {
		result[i] = account.MonetaryAccountBank
	}

	return result, nil
}

func (c *BunqClient) ensureSessionIsStarted() error {
	if c.session != nil {
		if _, err := c.doBunqRequest("GET", "/user", nil); err == nil {
			c.log.Debug("Reusing session from memory")
			return nil
		}

		// Session is no longer valid, start a new session
		c.log.Debug("Memory session is no longer valid")
		c.session = nil

		return c.startNewSession()
	}

	sessionServerLocation := c.config.StorageLocation + c.config.BunqConfig.SessionServerFileName
	if _, err := os.Stat(sessionServerLocation); err == nil {
		sessionServerJson, err := os.ReadFile(sessionServerLocation)
		if err != nil {
			c.log.WithError(err).Error("Cannot read session file from storage")
			return err
		}

		var sessionServer BunqSessionServer
		if err := json.Unmarshal(sessionServerJson, &sessionServer); err != nil {
			os.Remove(sessionServerLocation)
			c.log.WithError(err).Error("Cannot unmarshal stored session")
			return err
		}

		c.session = &sessionServer

		if _, err := c.doBunqRequest("GET", "/user", nil); err == nil {
			c.log.Debug("Reusing session from storage")
			return nil
		}

		// Session is no longer valid, start a new session
		c.log.Debug("Stored session is no longer valid")
		c.session = nil
	}

	return c.startNewSession()
}

func (c *BunqClient) startNewSession() error {
	c.log.Debug("Start new session")

	response, err := c.doBunqRequest("POST", "/session-server", BunqSessionServerRequest{
		Secret: c.config.BunqConfig.ApiKey,
	})
	if err != nil {
		c.log.WithError(err).Error("Starting new session failed")
		return err
	}

	var sessionServerResponse BunqSessionServerResponse
	if err := json.Unmarshal(response, &sessionServerResponse); err != nil {
		c.log.WithError(err).Error("Unmarshal session response failed")
		return err
	}

	sessionServer := BunqSessionServer{}
	for _, item := range sessionServerResponse.Response {
		if item.Id != nil {
			sessionServer.Id = item.Id
		}

		if item.Token != nil {
			sessionServer.Token = item.Token
		}

		if item.UserPerson != nil {
			sessionServer.UserPerson = item.UserPerson
		}
	}

	sessionServerJson, err := json.Marshal(sessionServer)
	if err != nil {
		c.log.WithError(err).Error("Marshal session server failed")
		return err
	}

	sessionServerLocation := c.config.StorageLocation + c.config.BunqConfig.SessionServerFileName
	if err := os.WriteFile(sessionServerLocation, sessionServerJson, 0700); err != nil {
		os.Remove(sessionServerLocation)
		c.log.WithError(err).Error("Cannot write session to storage")
		return err
	}
	c.log.Debug("Store session in storage")

	c.session = &sessionServer

	return nil
}

func (c *BunqClient) doBunqRequest(method string, path string, data interface{}) ([]byte, error) {
	requestId := uuid.New()
	log := c.log.WithField("requestId", requestId.String())

	body, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("Cannot marshal request body")
		return nil, err
	}

	url := c.config.BunqConfig.ApiBaseUrl + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.WithError(err).Error("Cannot create new request")
		return nil, err
	}
	req.Header.Set("User-Agent", c.config.BunqConfig.UserAgent)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Bunq-Client-Request-Id", requestId.String())

	if c.installation.Token != nil && c.session == nil {
		log.Debug("Use installation token for bunq authentication")
		req.Header.Set("X-Bunq-Client-Authentication", c.installation.Token.Token)
	}

	// Session token has higher priority than installation token
	if c.session != nil {
		log.Debug("Use session token for bunq authentication")
		req.Header.Set("X-Bunq-Client-Authentication", c.session.Token.Token)
	}

	if len(body) > 0 {
		base64Signature, err := c.keyChain.Sign(body)
		if err != nil {
			log.WithError(err).Error("Cannot sign request body")
			return nil, err
		}
		log.Debug("Generated signature for bunq request")
		req.Header.Set("X-Bunq-Client-Signature", base64Signature)
	}

	log.WithFields(
		logrus.Fields{
			"method": method,
			"path":   path,
		},
	).Debug("Send bunq request")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.WithError(err).Error("Cannot send request")
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Cannot read bunq response body")
		return nil, err
	}

	log.WithFields(
		logrus.Fields{
			"statusCode":        resp.StatusCode,
			"bodyLength":        len(respBody),
			"responseRequestId": resp.Header.Get("X-Bunq-Client-Request-Id"),
		},
	).Debug("Response received from bunq")

	if resp.Header.Get("X-Bunq-Client-Request-Id") != requestId.String() {
		log.Error("Received response for another request")
		return nil, errors.New("received response for another request")
	}

	// TODO: Make all calls verify the signature, for now only the session-server call works. All other calls fail for some reason
	if len(respBody) > 0 && c.bunqServerPublicKey != nil && path == "/session-server" {
		hashedBody := sha256.Sum256(respBody)
		serverSignature, err := base64.StdEncoding.DecodeString(resp.Header.Get("X-Bunq-Server-Signature"))
		if err != nil {
			log.WithError(err).Error("Cannot decode bunq server signature")
			return nil, err
		}

		if err := rsa.VerifyPKCS1v15(c.bunqServerPublicKey, crypto.SHA256, hashedBody[:], serverSignature); err != nil {
			log.WithError(err).Error("Cannot verify bunq server signature")
			return nil, err
		}

		log.Debug("Bunq server signature verified successfully")
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.WithField("body", respBody).Warn("Received error from bunq")
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}

func (c *BunqClient) parseBunqServerPublicKey() error {
	bunqPublicKey, _ := pem.Decode([]byte(c.installation.ServerPublicKey.ServicePublicKey))
	bunqServerPublicKey, err := x509.ParsePKIXPublicKey(bunqPublicKey.Bytes)
	if err != nil {
		c.log.WithError(err).Error("Cannot decode bunq server public key")
		return err
	}
	c.bunqServerPublicKey = bunqServerPublicKey.(*rsa.PublicKey)

	return nil
}
