package main

import (
	"errors"
	"os"
	"strings"
)

type BunqConfig struct {
	ApiBaseUrl           string
	ApiKey               string
	PrivateKeyFileName   string
	PublicKeyFileName    string
	InstallationFileName string
	DeviceServerFileName string
	UserAgent            string
	PermittedIps         []string
}

type Config struct {
	BunqConfig      *BunqConfig
	StorageLocation string
}

func LoadConfig() (*Config, error) {
	storageLocation, exists := os.LookupEnv("STORAGE_LOCATION")
	if !exists {
		storageLocation = "./storage/"
	}
	if string(storageLocation[len(storageLocation)-1]) != "/" {
		return nil, errors.New("storage location must end with a slash")
	}

	bunqConfig, err := loadBunqConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		StorageLocation: storageLocation,
		BunqConfig:      bunqConfig,
	}, nil
}

func loadBunqConfig() (*BunqConfig, error) {
	apiBaseUrl, exists := os.LookupEnv("BUNQ_API_BASE_URL")
	if !exists {
		apiBaseUrl = "https://public-api.sandbox.bunq.com/v1"
	}
	if string(apiBaseUrl[len(apiBaseUrl)-1]) == "/" {
		return nil, errors.New("bunq api base url cannot end with a slash")
	}

	apiKey, exists := os.LookupEnv("BUNQ_API_KEY")
	if !exists {
		return nil, errors.New("missing bunq api key in env")
	}

	privateKeyFileName, exists := os.LookupEnv("BUNQ_PRIVATE_KEY_FILE_NAME")
	if !exists {
		privateKeyFileName = "bunq_client.key"
	}

	publicKeyFileName, exists := os.LookupEnv("BUNQ_PUBLIC_KEY_FILE_NAME")
	if !exists {
		publicKeyFileName = "bunq_client.pub.key"
	}

	installationFileName, exists := os.LookupEnv("BUNQ_INSTALLATION_FILE_NAME")
	if !exists {
		installationFileName = "bunq_installation.json"
	}

	deviceServerFileName, exists := os.LookupEnv("BUNQ_DEVICE_SERVER_FILE_NAME")
	if !exists {
		deviceServerFileName = "bunq_device_server.json"
	}

	userAgent, exists := os.LookupEnv("BUNQ_USER_AGENT")
	if !exists {
		userAgent = "BunqFireflySync/1.0"
	}

	permittedIps, exists := os.LookupEnv("BUNQ_PERMITTED_IPS")
	if !exists {
		permittedIps = "*"
	}
	permittedIpsSplitted := strings.Split(permittedIps, ",")

	return &BunqConfig{
		ApiBaseUrl:           apiBaseUrl,
		ApiKey:               apiKey,
		PrivateKeyFileName:   privateKeyFileName,
		PublicKeyFileName:    publicKeyFileName,
		InstallationFileName: installationFileName,
		DeviceServerFileName: deviceServerFileName,
		UserAgent:            userAgent,
		PermittedIps:         permittedIpsSplitted,
	}, nil
}
