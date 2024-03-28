package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
)

type Keychain struct {
	privateKeyPath string
	publicKeyPath  string
	PrivateKey     *rsa.PrivateKey
	PublicKey      *rsa.PublicKey
	PrivateKeyPem  []byte
	PublicKeyPem   []byte
	bitSize        int
}

func NewKeyChain(privateKeyPath string, publicKeyPath string, bitSize int) (*Keychain, error) {
	instance := Keychain{
		privateKeyPath: privateKeyPath,
		publicKeyPath:  publicKeyPath,
		bitSize:        bitSize,
	}

	if err := instance.loadOrCreateKeypair(); err != nil {
		return nil, err
	}

	return &instance, nil
}

func (kc *Keychain) loadOrCreateKeypair() error {
	if kc.privateKeyPath != "" {
		if _, err := os.Stat(kc.privateKeyPath); err == nil {
			if err := kc.loadPrivateKey(); err != nil {
				return err
			}
		} else {
			if err := kc.createPrivateKey(); err != nil {
				return err
			}
		}
	}

	if kc.publicKeyPath != "" {
		if _, err := os.Stat(kc.publicKeyPath); err == nil {
			if err := kc.loadPublicKey(); err != nil {
				return err
			}
		} else {
			if err := kc.createPublicKey(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (kc *Keychain) loadPrivateKey() error {
	privateKey, err := os.ReadFile(kc.privateKeyPath)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(privateKey)
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	kc.PrivateKey = key.(*rsa.PrivateKey)
	kc.PrivateKeyPem = privateKey

	return nil
}

func (kc *Keychain) loadPublicKey() error {
	publicKey, err := os.ReadFile(kc.publicKeyPath)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(publicKey)
	key, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	kc.PublicKey = key.(*rsa.PublicKey)
	kc.PublicKeyPem = publicKey

	return nil
}

func (kc *Keychain) createPrivateKey() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, kc.bitSize)
	if err != nil {
		return err
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	privateKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})

	if err := os.WriteFile(kc.privateKeyPath, privateKeyPem, 0700); err != nil {
		return err
	}

	kc.PrivateKey = privateKey
	kc.PrivateKeyPem = privateKeyPem

	return nil
}

func (kc *Keychain) createPublicKey() error {
	if kc.PrivateKey == nil {
		return errors.New("cannot create public key, private key is missing in the keychain")
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(kc.PrivateKey.Public())
	if err != nil {
		return err
	}

	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	if err := os.WriteFile(kc.publicKeyPath, publicKeyPem, 0700); err != nil {
		return err
	}

	kc.PublicKey = &kc.PrivateKey.PublicKey
	kc.PublicKeyPem = publicKeyPem

	return nil
}

func (kc *Keychain) Sign(data []byte) (string, error) {
	hashedBody := sha256.Sum256(data)

	signature, err := rsa.SignPKCS1v15(rand.Reader, kc.PrivateKey, crypto.SHA256, hashedBody[:])
	if err != nil {
		return "", err
	}

	base64Signature := base64.StdEncoding.EncodeToString(signature)
	return base64Signature, nil
}
