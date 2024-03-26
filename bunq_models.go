package main

type BunqInstallationRequest struct {
	ClientPublicKey string `json:"client_public_key"`
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
