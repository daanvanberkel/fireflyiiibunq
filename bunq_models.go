package main

// BUNQ INSTALLATION MODELS

type BunqInstallationRequest struct {
	ClientPublicKey string `json:"client_public_key"`
}

type BunqInstallationResponse struct {
	Response []*BunqInstallation `json:"Response"`
}

type BunqInstallation struct {
	Id              *BunqInstallationId              `json:"Id"`
	Token           *BunqInstallationToken           `json:"Token"`
	ServerPublicKey *BunqInstallationServerPublicKey `json:"ServerPublicKey"`
}

type BunqInstallationId struct {
	Id int32 `json:"id"`
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
	Id *BunqDeviceServerId `json:"Id"`
}

type BunqDeviceServerId struct {
	Id int32 `json:"id"`
}
