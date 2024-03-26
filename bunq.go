package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
)

type BunqClient struct {
	apiBaseUrl           string
	privateKeyLocation   string
	publicKeyLocation    string
	installationLocation string
	httpClient           *http.Client
	installation         BunqInstallation
}

func NewBunqClient() BunqClient {
	// TODO: Get storage location from env var or default location
	return BunqClient{
		apiBaseUrl:           "https://public-api.sandbox.bunq.com/v1",
		privateKeyLocation:   "key.rsa",
		publicKeyLocation:    "key.rsa.pub",
		installationLocation: "installation.json",
		httpClient:           &http.Client{},
	}
}

func (c *BunqClient) LoadInstallation() error {

	// Try to load installation from storage
	if _, err := os.Stat(c.installationLocation); err == nil {
		data, err := os.ReadFile(c.installationLocation)
		if err != nil {
			return err
		}

		var installation BunqInstallation
		if err := json.Unmarshal(data, &installation); err != nil {
			return err
		}

		c.installation = installation
		return nil
	}

	// Generate new installation
	_, publicKey, err := c.getKeyPair()
	if err != nil {
		return err
	}

	response, err := c.doBunqRequest("POST", "/installation", BunqInstallationRequest{
		ClientPublicKey: publicKey,
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

	if err := os.WriteFile(c.installationLocation, installationJson, 0700); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) getKeyPair() (string, string, error) {
	keyFileName := c.privateKeyLocation
	pubKeyFileName := c.publicKeyLocation
	bitSize := 2048

	if _, err := os.Stat(keyFileName); err == nil {
		if _, err := os.Stat(pubKeyFileName); err == nil {
			privateKey, err := os.ReadFile(keyFileName)
			if err != nil {
				return "", "", err
			}

			publicKey, err := os.ReadFile(pubKeyFileName)
			if err != nil {
				return "", "", err
			}

			return string(privateKey), string(publicKey), nil
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return "", "", err
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return "", "", err
	}

	privateKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})
	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})

	if err := os.WriteFile(keyFileName, privateKeyPem, 0700); err != nil {
		return "", "", err
	}

	if err := os.WriteFile(pubKeyFileName, publicKeyPem, 0700); err != nil {
		return "", "", err
	}

	return string(privateKeyPem), string(publicKeyPem), nil
}

func (c *BunqClient) doBunqRequest(method string, path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := c.apiBaseUrl + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FireflyIIIBunqSync/1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New(string(respBody))
	}
	return respBody, nil
}
