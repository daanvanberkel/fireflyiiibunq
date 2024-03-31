package bunq

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

type BunqSession struct {
	apiKey          string
	sessionLocation string
	sessionServer   *BunqSessionServer
	client          *BunqHttpClient
	log             *logrus.Logger
}

func NewBunqSession(apiKey string, sessionLocation string, client *BunqHttpClient, log *logrus.Logger) (*BunqSession, error) {
	session := &BunqSession{
		apiKey:          apiKey,
		sessionLocation: sessionLocation,
		client:          client,
		log:             log,
	}

	if err := session.loadSession(); err != nil {
		if err := session.StartSession(); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (s *BunqSession) GetToken() (string, error) {
	if s.sessionServer == nil || s.sessionServer.Token.Token == "" {
		return "", errors.New("no session token in storage")
	}

	return s.sessionServer.Token.Token, nil
}

func (s *BunqSession) GetUserId() (int, error) {
	if s.sessionServer == nil || s.sessionServer.UserPerson == nil {
		return -1, errors.New("no user id in storage")
	}

	return s.sessionServer.UserPerson.Id, nil
}

func (s *BunqSession) StartSession() error {
	s.log.Debug("Start new session")

	response, err := s.client.DoBunqRequest("POST", "/session-server", BunqSessionServerRequest{
		Secret: s.apiKey,
	})
	if err != nil {
		s.log.WithError(err).Error("Starting new session failed")
		return err
	}

	var sessionServerResponse BunqSessionServerResponse
	if err := json.Unmarshal(response, &sessionServerResponse); err != nil {
		s.log.WithError(err).Error("Unmarshal session response failed")
		return err
	}

	sessionServer := BunqSessionServer{}
	for _, item := range sessionServerResponse.Response {
		if item.Id != nil {
			sessionServer.Id = item.Id
		}

		if item.Token != nil {
			sessionServer.Token = item.Token
		}

		if item.UserPerson != nil {
			sessionServer.UserPerson = item.UserPerson
		}
	}

	if err := s.writeSessionToFile(&sessionServer); err != nil {
		return err
	}

	s.sessionServer = &sessionServer
	return nil
}

func (s *BunqSession) loadSession() error {
	sessionServer, err := s.readSessionFromFile()
	if err != nil {
		return err
	}

	s.sessionServer = sessionServer
	s.client.SetSession(s)
	return nil
}

func (s *BunqSession) writeSessionToFile(sessionServer *BunqSessionServer) error {
	sessionServerJson, err := json.Marshal(sessionServer)
	if err != nil {
		s.log.WithError(err).Error("Marshal session server failed")
		return err
	}

	if err := os.WriteFile(s.sessionLocation, sessionServerJson, 0700); err != nil {
		os.Remove(s.sessionLocation)
		s.log.WithError(err).Error("Cannot write session to storage")
		return err
	}
	s.log.Debug("Store session in storage")

	return nil
}

func (s *BunqSession) readSessionFromFile() (*BunqSessionServer, error) {
	if _, err := os.Stat(s.sessionLocation); err != nil {
		return nil, errors.New("cannot load session from storage, session file not found")
	}

	sessionServerJson, err := os.ReadFile(s.sessionLocation)
	if err != nil {
		s.log.WithError(err).Error("Cannot read session file from storage")
		return nil, err
	}

	var sessionServer BunqSessionServer
	if err := json.Unmarshal(sessionServerJson, &sessionServer); err != nil {
		os.Remove(s.sessionLocation)
		s.log.WithError(err).Error("Cannot unmarshal stored session")
		return nil, err
	}

	return &sessionServer, nil
}
