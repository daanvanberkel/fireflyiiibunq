package main

type BunqInstallationRequest struct {
	ClientPublicKey []byte `json:"client_public_key"`
}

type BunqInstallation struct {
	Token           *BunqInstallationToken           `json:"Token"`
	ServerPublicKey *BunqInstallationServerPublicKey `json:"ServerPublicKey"`
}

type BunqInstallationToken struct {
	Token string `json:"token"`
}

type BunqInstallationServerPublicKey struct {
	ServicePublicKey string `json:"server_public_key"`
}
