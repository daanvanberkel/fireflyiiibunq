package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
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
	apiBaseUrl           string
	apiKey               string
	privateKeyLocation   string
	publicKeyLocation    string
	installationLocation string
	deviceServerLocation string
	httpClient           *http.Client
	installation         BunqInstallation
	privateKey           *rsa.PrivateKey
	bunqServerPublicKey  *rsa.PublicKey
}

func NewBunqClient(apiKey string) BunqClient {
	// TODO: Get storage location from env var or default location
	return BunqClient{
		apiBaseUrl:           "https://public-api.sandbox.bunq.com/v1",
		apiKey:               apiKey,
		privateKeyLocation:   "storage/key.rsa",
		publicKeyLocation:    "storage/key.rsa.pub",
		installationLocation: "storage/installation.json",
		deviceServerLocation: "storage/device-server.json",
		httpClient:           &http.Client{},
	}
}

func (c *BunqClient) LoadInstallation() error {
	_, publicKey, err := c.getKeyPair()
	if err != nil {
		return err
	}

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

		if err := c.parseBunqServerPublicKey(); err != nil {
			return err
		}

		return nil
	}

	// Generate new installation
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

	if err := c.parseBunqServerPublicKey(); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) LoadDeviceServer() error {
	if _, err := os.Stat(c.deviceServerLocation); err == nil {
		return nil
	}

	response, err := c.doBunqRequest("POST", "/device-server", BunqDeviceServerRequest{
		Description:  "FireFlyIIIBynqSync",
		Secret:       c.apiKey,
		PermittedIps: []string{"*"}, // TODO: Make permitted ips configurable
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

	if err := os.WriteFile(c.deviceServerLocation, deviceServerJson, 0700); err != nil {
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

			keyBlock, _ := pem.Decode(privateKey)
			key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
			if err != nil {
				return "", "", err
			}

			c.privateKey = key.(*rsa.PrivateKey)

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

	c.privateKey = privateKey

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

	if c.installation.Token != nil {
		req.Header.Set("X-Bunq-Client-Authentication", c.installation.Token.Token)
	}

	if len(body) > 0 && c.privateKey != nil {
		hashedBody := sha256.Sum256(body)

		signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashedBody[:])
		if err != nil {
			return nil, err
		}

		base64Signature := base64.StdEncoding.EncodeToString(signature)
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
