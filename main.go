package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Out = os.Stdout

	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	client, err := NewBunqClient(config, log)
	if err != nil {
		panic(err)
	}
	if err := client.LoadInstallation(); err != nil {
		panic(err)
	}
	if err := client.LoadDeviceServer(); err != nil {
		panic(err)
	}

	bankAccounts, err := client.GetMonetaryBankAccounts()
	if err != nil {
		panic(err)
	}
	fmt.Println(json.Marshal(bankAccounts))
}
