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

	var installation BunqInstallation
	if err := json.Unmarshal(response, &installation); err != nil {
		return err
	}
	c.installation = installation

	return nil
}

func (c *BunqClient) getKeyPair() ([]byte, []byte, error) {
	keyFileName := c.privateKeyLocation
	pubKeyFileName := c.publicKeyLocation
	bitSize := 2048

	// TODO: Get storage location from env var or default location
	if _, err := os.Stat(keyFileName); err == nil {
		if _, err := os.Stat(pubKeyFileName); err == nil {
			privateKey, err := os.ReadFile(keyFileName)
			if err != nil {
				return nil, nil, err
			}

			publicKey, err := os.ReadFile(pubKeyFileName)
			if err != nil {
				return nil, nil, err
			}

			return privateKey, publicKey, nil
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, nil, err
	}

	publicKey := privateKey.Public()

	privateKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(publicKey.(*rsa.PublicKey))})

	if err := os.WriteFile(keyFileName, privateKeyPem, 0700); err != nil {
		return nil, nil, err
	}

	if err := os.WriteFile(pubKeyFileName, publicKeyPem, 0700); err != nil {
		return nil, nil, err
	}

	return privateKeyPem, publicKeyPem, nil
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
