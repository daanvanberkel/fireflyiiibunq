package firefly

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

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
	}).Debug("Response received from firefly")

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.WithField("body", respBody).Warn("Received error from bunq")
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}
