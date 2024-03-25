package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	privateKey, publicKey, err := getKeyPair()
	if err != nil {
		panic(err)
	}

	fmt.Println(privateKey)
	fmt.Println(publicKey)
}

func getKeyPair() ([]byte, []byte, error) {
	keyFileName := "key.rsa"
	pubKeyFileName := keyFileName + ".pub"
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
