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
)

type BunqClient struct {
	config              *Config
	httpClient          *http.Client
	installation        BunqInstallation
	keyChain            *Keychain
	bunqServerPublicKey *rsa.PublicKey
}

func NewBunqClient(config *Config) (*BunqClient, error) {
	keyChain, err := NewKeyChain(config.StorageLocation+config.BunqConfig.PrivateKeyFileName, config.StorageLocation+config.BunqConfig.PublicKeyFileName, 2048)
	if err != nil {
		return nil, err
	}

	return &BunqClient{
		config:     config,
		httpClient: &http.Client{},
		keyChain:   keyChain,
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

func (c *BunqClient) doBunqRequest(method string, path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := c.config.BunqConfig.ApiBaseUrl + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FireflyIIIBunqSync/1")

	if c.installation.Token != nil {
		req.Header.Set("X-Bunq-Client-Authentication", c.installation.Token.Token)
	}

	if len(body) > 0 {
		base64Signature, err := c.keyChain.Sign(body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Bunq-Client-Signature", base64Signature)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(respBody) > 0 && c.bunqServerPublicKey != nil {
		hashedBody := sha256.Sum256(respBody)
		serverSignature, err := base64.StdEncoding.DecodeString(resp.Header.Get("X-Bunq-Server-Signature"))
		if err != nil {
			return nil, err
		}

		if err := rsa.VerifyPKCS1v15(c.bunqServerPublicKey, crypto.SHA256, hashedBody[:], serverSignature); err != nil {
			return nil, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}

func (c *BunqClient) parseBunqServerPublicKey() error {
	bunqPublicKey, _ := pem.Decode([]byte(c.installation.ServerPublicKey.ServicePublicKey))
	bunqServerPublicKey, err := x509.ParsePKIXPublicKey(bunqPublicKey.Bytes)
	if err != nil {
		return err
	}
	c.bunqServerPublicKey = bunqServerPublicKey.(*rsa.PublicKey)

	return nil
}
