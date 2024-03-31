package bunq

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/sirupsen/logrus"
)

type BunqClient struct {
	config   *util.Config
	session  *BunqSession
	client   *BunqHttpClient
	keyChain *util.Keychain
	log      *logrus.Logger
}

func NewBunqClient(config *util.Config, log *logrus.Logger) (*BunqClient, error) {
	keyChain, err := util.NewKeyChain(config.StorageLocation+config.BunqConfig.PrivateKeyFileName, config.StorageLocation+config.BunqConfig.PublicKeyFileName, 2048)
	if err != nil {
		return nil, err
	}

	httpClient, err := NewBunqHttpClient(config.BunqConfig.ApiBaseUrl, config.BunqConfig.UserAgent, log)
	if err != nil {
		return nil, err
	}
	httpClient.SetKeyChain(keyChain)

	client := &BunqClient{
		config:   config,
		keyChain: keyChain,
		client:   httpClient,
		log:      log,
	}

	if err := client.boot(); err != nil {
		return nil, err
	}
	return client, nil
}

// ENDPOINT CALLS

func (c *BunqClient) GetMonetaryBankAccounts() ([]*BunqMonetaryAccountBank, error) {
	if err := c.startSession(); err != nil {
		return nil, err
	}

	userId, err := c.session.GetUserId()
	if err != nil {
		return nil, err
	}

	response, err := c.client.DoBunqRequest("GET", "/user/"+strconv.Itoa(userId)+"/monetary-account-bank", nil)
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

func (c *BunqClient) GetPayments(monetaryAccountId int, olderThanId int) ([]*BunqPayment, error) {
	if err := c.startSession(); err != nil {
		return nil, err
	}

	userId, err := c.session.GetUserId()
	if err != nil {
		return nil, err
	}

	url := "/user/" + strconv.Itoa(userId) + "/monetary-account/" + strconv.Itoa(monetaryAccountId) + "/payment"
	if olderThanId > 0 {
		url += "?older_id=" + strconv.Itoa(olderThanId)
	}

	response, err := c.client.DoBunqRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var paymentResponse BunqPaymentsResponse
	if err := json.Unmarshal(response, &paymentResponse); err != nil {
		return nil, err
	}

	result := make([]*BunqPayment, len(paymentResponse.Response))
	for i, payment := range paymentResponse.Response {
		result[i] = payment.Payment
	}

	return result, nil
}

// UTILS

func (c *BunqClient) boot() error {
	if err := c.loadInstallation(); err != nil {
		return err
	}

	if err := c.loadDeviceServer(); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) loadInstallation() error {
	// Try to load installation from storage
	installationPath := c.config.StorageLocation + c.config.BunqConfig.InstallationFileName
	if _, err := os.Stat(installationPath); err == nil {
		data, err := os.ReadFile(installationPath)
		if err != nil {
			return err
		}

		var installation BunqInstallationServer
		if err := json.Unmarshal(data, &installation); err != nil {
			return err
		}

		c.client.SetInstallation(&installation)

		return nil
	}

	// Generate new installation
	response, err := c.client.DoBunqRequest("POST", "/installation", BunqInstallationRequest{
		ClientPublicKey: string(c.keyChain.PublicKeyPem),
	})
	if err != nil {
		return err
	}

	var installationResponse BunqInstallationResponse
	if err := json.Unmarshal(response, &installationResponse); err != nil {
		return err
	}

	installation := BunqInstallationServer{}
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

	c.client.SetInstallation(&installation)
	installationJson, err := json.Marshal(installation)
	if err != nil {
		return err
	}

	if err := os.WriteFile(installationPath, installationJson, 0700); err != nil {
		return err
	}

	return nil
}

func (c *BunqClient) loadDeviceServer() error {
	deviceServerPath := c.config.StorageLocation + c.config.BunqConfig.DeviceServerFileName
	if _, err := os.Stat(deviceServerPath); err == nil {
		return nil
	}

	response, err := c.client.DoBunqRequest("POST", "/device-server", BunqDeviceServerRequest{
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

func (c *BunqClient) startSession() error {
	if c.session != nil {
		return nil
	}

	session, err := NewBunqSession(c.config.BunqConfig.ApiKey, c.config.StorageLocation+c.config.BunqConfig.SessionServerFileName, c.client, c.log)
	if err != nil {
		return err
	}
	c.session = session

	return nil
}
