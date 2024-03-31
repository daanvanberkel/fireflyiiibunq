package bunq

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// BUNQ COMMON MODELS
type BunqId struct {
	Id int `json:"id"`
}

type BunqUserPerson struct {
	Id int `json:"id"`
}

type BunqAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type BunqPointer struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Name  string `json:"name"`
}

type BunqPagination struct {
	FutureUrl string `json:"future_url"`
	NewerUrl  string `json:"newer_url"`
	OlderUrl  string `json:"older_url"`
}

// BUNQ INSTALLATION MODELS

type BunqInstallationRequest struct {
	ClientPublicKey string `json:"client_public_key"`
}

type BunqInstallationResponse struct {
	Response []*BunqInstallationServer `json:"Response"`
}

type BunqInstallationServer struct {
	Id              *BunqId                          `json:"Id"`
	Token           *BunqInstallationToken           `json:"Token"`
	ServerPublicKey *BunqInstallationServerPublicKey `json:"ServerPublicKey"`
}

func (i *BunqInstallationServer) GetServerPublicKey() (*rsa.PublicKey, error) {
	bunqPublicKey, _ := pem.Decode([]byte(i.ServerPublicKey.ServicePublicKey))
	bunqServerPublicKey, err := x509.ParsePKIXPublicKey(bunqPublicKey.Bytes)
	if err != nil {
		return nil, err
	}
	return bunqServerPublicKey.(*rsa.PublicKey), nil
}

type BunqInstallationToken struct {
	Id      int    `json:"id"`
	Created string `json:"created"`
	Updated string `json:"updated"`
	Token   string `json:"token"`
}

type BunqInstallationServerPublicKey struct {
	ServicePublicKey string `json:"server_public_key"`
}

// BUNQ DEVICE SERVER MODELS

type BunqDeviceServerRequest struct {
	Description  string   `json:"description"`
	Secret       string   `json:"secret"`
	PermittedIps []string `json:"permitted_ips"`
}

type BunqDeviceServerResponse struct {
	Response []*BunqDeviceServer `json:"Response"`
}

type BunqDeviceServer struct {
	Id *BunqId `json:"Id"`
}

// BUNQ SESSION SERVER MODELS

type BunqSessionServerRequest struct {
	Secret string `json:"secret"`
}

type BunqSessionServerResponse struct {
	Response []*BunqSessionServer `json:"Response"`
}

type BunqSessionServer struct {
	Id         *BunqId                 `json:"Id"`
	Token      *BunqSessionServerToken `json:"Token"`
	UserPerson *BunqUserPerson         `json:"UserPerson"`
}

type BunqSessionServerToken struct {
	Id    int    `json:"id"`
	Token string `json:"token"`
}

// BUNQ MONETARY ACCOUNT BANK MODELS

type BunqMonetaryAccountBankResponse struct {
	Response []*BunqMonetaryAccountBankItem `json:"Response"`
}

type BunqMonetaryAccountBankItem struct {
	MonetaryAccountBank *BunqMonetaryAccountBank `json:"MonetaryAccountBank"`
}

type BunqMonetaryAccountBank struct {
	Id                int            `json:"id"`
	Created           string         `json:"created"`
	Updated           string         `json:"updated"`
	Currency          string         `json:"currency"`
	Description       string         `json:"description"`
	Status            string         `json:"status"`
	SubStatus         string         `json:"sub_status"`
	Reason            string         `json:"reason"`
	ReasonDescription string         `json:"reason_description"`
	DisplayName       string         `json:"display_name"`
	DailyLimit        *BunqAmount    `json:"daily_limit"`
	OverdraftLimit    *BunqAmount    `json:"overdraft_limit"`
	Balance           *BunqAmount    `json:"balance"`
	PublicUuid        string         `json:"public_uuid"`
	UserId            int            `json:"user_id"`
	Alias             []*BunqPointer `json:"alias"`
}

func (account *BunqMonetaryAccountBank) GetIBAN() (string, error) {
	for _, alias := range account.Alias {
		if alias.Type == "IBAN" {
			return alias.Value, nil
		}
	}

	return "", errors.New("iban not found")
}

// BUNQ PAYMENT MODELS

type BunqPaymentsResponse struct {
	Response   []*BunqPaymentResponse `json:"Response"`
	Pagination *BunqPagination        `json:"Pagination"`
}

type BunqPaymentResponse struct {
	Payment *BunqPayment `json:"Payment"`
}

type BunqPayment struct {
	Id                   int                         `json:"id"`
	Created              string                      `json:"created"`
	Updated              string                      `json:"updated"`
	MonetaryAccountId    int                         `json:"monetary_account_id"`
	Amount               *BunqAmount                 `json:"amount"`
	Description          string                      `json:"description"`
	Type                 string                      `json:"type"`
	MerchantReference    string                      `json:"merchant_reference"`
	BalanceAfterMutation *BunqAmount                 `json:"balance_after_mutation"`
	Alias                *BunqPaymentMonetaryAccount `json:"alias"`
	CounterpartyAlias    *BunqPaymentMonetaryAccount `json:"counterparty_alias"`
}

type BunqPaymentMonetaryAccount struct {
	Iban        string `json:"iban"`
	DisplayName string `json:"display_name"`
	Country     string `json:"country"`
}
