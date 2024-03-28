package main

// BUNQ COMMON MODELS
type BunqId struct {
	Id int32 `json:"id"`
}

type BunqUserPerson struct {
	Id int `json:"id"`
}

// BUNQ INSTALLATION MODELS

type BunqInstallationRequest struct {
	ClientPublicKey string `json:"client_public_key"`
}

type BunqInstallationResponse struct {
	Response []*BunqInstallation `json:"Response"`
}

type BunqInstallation struct {
	Id              *BunqId                          `json:"Id"`
	Token           *BunqInstallationToken           `json:"Token"`
	ServerPublicKey *BunqInstallationServerPublicKey `json:"ServerPublicKey"`
}

type BunqInstallationToken struct {
	Id      int32  `json:"id"`
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
	Id    int32  `json:"id"`
	Token string `json:"token"`
}

// BUNQ MONETARY ACCOUNT BANK MODELS

type BunqMonetaryAccountBankResponse struct {
	Response []*BunqMonetaryAccountBank `json:"Response"`
}

type BunqMonetaryAccountBank struct {
	Id          int32  `json:"id"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}
