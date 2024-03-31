package bunq

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type BunqHttpClient struct {
	apiBaseUrl   string
	userAgent    string
	log          *logrus.Logger
	installation *BunqInstallationServer
	session      *BunqSession
	keyChain     *util.Keychain
	httpClient   *http.Client
	maxRetries   int
}

func NewBunqHttpClient(apiBaseUrl string, userAgent string, log *logrus.Logger) (*BunqHttpClient, error) {
	return &BunqHttpClient{
		apiBaseUrl: apiBaseUrl,
		userAgent:  userAgent,
		log:        log,
		httpClient: &http.Client{},
		maxRetries: 3,
	}, nil
}

func (c *BunqHttpClient) SetInstallation(installation *BunqInstallationServer) {
	c.installation = installation
}

func (c *BunqHttpClient) SetSession(session *BunqSession) {
	c.session = session
}

func (c *BunqHttpClient) SetKeyChain(keyChain *util.Keychain) {
	c.keyChain = keyChain
}

func (c *BunqHttpClient) DoBunqRequest(method string, path string, data interface{}) ([]byte, error) {
	return c.doActualBunqRequest(method, path, data, 1)
}

func (c *BunqHttpClient) doActualBunqRequest(method string, path string, data interface{}, try int) ([]byte, error) {
	requestId := uuid.New()
	log := c.log.WithFields(logrus.Fields{
		"requestId": requestId.String(),
		"try":       try,
		"method":    method,
		"path":      path,
	})

	if try > c.maxRetries {
		c.log.Error("Max retires reached")
		return nil, errors.New("max retries reached")
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("Cannot marshal request body")
		return nil, err
	}

	url := c.apiBaseUrl + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.WithError(err).Error("Cannot create new request")
		return nil, err
	}

	if err := c.setDefaultHeaders(req, &requestId, log); err != nil {
		return nil, err
	}
	if err := c.signRequestBody(req, body, log); err != nil {
		return nil, err
	}

	log.Debug("Send bunq request")
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

	if err := c.validateServerResponseBody(resp, respBody, path, log); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if (resp.StatusCode == 401 || resp.StatusCode == 403) && c.session != nil {
			if err := c.session.StartSession(); err != nil {
				return nil, err
			}

			log.Info("Received 401 or 403 from bunq, possible session expiry. Retry request")
			return c.doActualBunqRequest(method, path, data, try+1)
		}

		log.WithField("body", respBody).Warn("Received error from bunq")
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}

func (c *BunqHttpClient) setDefaultHeaders(request *http.Request, requestId *uuid.UUID, log *logrus.Entry) error {
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("X-Bunq-Client-Request-Id", requestId.String())

	if c.installation != nil && c.installation.Token != nil && c.session == nil {
		log.Debug("Use installation token for bunq authentication")
		request.Header.Set("X-Bunq-Client-Authentication", c.installation.Token.Token)
	}

	// Session token has higher priority than installation token
	if c.session != nil {
		sessionToken, err := c.session.GetToken()
		if err != nil {
			return err
		}

		log.Debug("Use session token for bunq authentication")
		request.Header.Set("X-Bunq-Client-Authentication", sessionToken)
	}

	if c.installation == nil && c.session == nil {
		log.Info("Both installation and session tokens are missing, continuing without authentication")
	}

	return nil
}

func (c *BunqHttpClient) signRequestBody(request *http.Request, body []byte, log *logrus.Entry) error {
	if c.keyChain != nil && len(body) > 0 {
		base64Signature, err := c.keyChain.Sign(body)
		if err != nil {
			log.WithError(err).Error("Cannot sign request body")
			return err
		}
		log.Debug("Generated signature for bunq request")
		request.Header.Set("X-Bunq-Client-Signature", base64Signature)
	}

	if c.keyChain == nil {
		log.Info("Keychain missing, continuing without signing the request")
	}

	return nil
}

func (c *BunqHttpClient) validateServerResponseBody(response *http.Response, responseBody []byte, path string, log *logrus.Entry) error {
	// TODO: Make all calls verify the signature, for now only the session-server call works. All other calls fail for some reason
	if len(responseBody) > 0 && c.installation != nil && path == "/session-server" {
		hashedBody := sha256.Sum256(responseBody)
		serverSignature, err := base64.StdEncoding.DecodeString(response.Header.Get("X-Bunq-Server-Signature"))
		if err != nil {
			log.WithError(err).Error("Cannot decode bunq server signature")
			return err
		}

		publicKey, err := c.installation.GetServerPublicKey()
		if err != nil {
			log.WithError(err).Error("Cannot load bunq server public key")
			return err
		}

		if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashedBody[:], serverSignature); err != nil {
			log.WithError(err).Error("Cannot verify bunq server signature")
			return err
		}

		log.Debug("Bunq server signature verified successfully")
	}

	if c.installation == nil {
		log.Info("Bunq installation missing, continuing without server signature validation")
	}

	return nil
}
