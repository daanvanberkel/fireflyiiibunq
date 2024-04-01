package firefly

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type FireflyClient struct {
	client     *http.Client
	apiBaseUrl string
	apiKey     string
	log        *logrus.Logger
}

func NewFireflyClient(config *util.Config, log *logrus.Logger) (*FireflyClient, error) {
	return &FireflyClient{
		client:     &http.Client{},
		apiBaseUrl: config.FireflyConfig.ApiBaseUrl,
		apiKey:     config.FireflyConfig.ApiKey,
		log:        log,
	}, nil
}

func (c *FireflyClient) SearchAccounts(query string, field AccountField, accountType AccountType, page int) (*AccountsResponse, error) {
	queryParams := url.Values{
		"page":  {strconv.Itoa(page)},
		"query": {query},
		"type":  {string(accountType)},
		"field": {string(field)},
	}
	response, err := c.doFireflyRequest("GET", "/v1/search/accounts?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var accounts AccountsResponse
	if err := json.Unmarshal(response, &accounts); err != nil {
		return nil, err
	}

	return &accounts, nil
}

func (c *FireflyClient) CreateAccount(account *AccountRequest) (*AccountRead, error) {
	response, err := c.doFireflyRequest("POST", "/v1/accounts", account)
	if err != nil {
		return nil, err
	}

	var accountResponse AccountResponse
	if err := json.Unmarshal(response, &accountResponse); err != nil {
		return nil, err
	}

	return accountResponse.Data, nil
}

func (c *FireflyClient) FindOrCreateAssetAccount(iban string, request *AccountRequest) (*AccountRead, error) {
	accounts, err := c.SearchAccounts(iban, IbanField, AssetType, 1)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts.Data {
		if account.Attributes.AccountRole == DefaultAsset {
			c.log.WithField("iban", iban).Info("Found existing account")
			return account, nil
		}
	}

	// Account not found, try to create new account
	account, err := c.CreateAccount(request)
	if err != nil {
		return nil, err
	}
	c.log.WithField("iban", iban).Info("Created new firefly account")

	return account, nil
}

func (c *FireflyClient) SearchTransactions(query *TransactionSearchQuery, page int) (*TransactionsResponse, error) {
	queryParams := url.Values{
		"page":  {strconv.Itoa(page)},
		"query": {query.Encode()},
	}
	response, err := c.doFireflyRequest("GET", "/v1/search/transactions?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result TransactionsResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *FireflyClient) CreateTransaction(transaction *TransactionRequest) (*TransactionResponse, error) {
	response, err := c.doFireflyRequest("POST", "/v1/transactions", transaction)
	if err != nil {
		return nil, err
	}

	var transactionResponse TransactionResponse
	if err := json.Unmarshal(response, &transactionResponse); err != nil {
		return nil, err
	}

	return &transactionResponse, nil
}

func (c *FireflyClient) doFireflyRequest(method string, path string, data interface{}) ([]byte, error) {
	requestId := uuid.New()
	log := c.log.WithFields(logrus.Fields{
		"method":    method,
		"path":      path,
		"requestId": requestId.String(),
	})

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

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Trace-Id", requestId.String())
	req.Header.Set("Accept", "application/json")

	log.Debug("Send firefly request")
	resp, err := c.client.Do(req)
	if err != nil {
		log.WithError(err).Error("Cannot send request")
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Cannot read firefly response body")
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"statusCode": resp.StatusCode,
		"bodyLength": len(respBody),
	}).Info("Response received from firefly")

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.WithField("body", string(respBody)).Warn("Received error from firefly")
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}
