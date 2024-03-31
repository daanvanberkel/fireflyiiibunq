package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daanvanberkel/fireflyiiibunq/bunq"
	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Out = os.Stdout

	config, err := util.LoadConfig()
	if err != nil {
		panic(err)
	}

	client, err := bunq.NewBunqClient(config, log)
	if err != nil {
		panic(err)
	}

	bankAccounts, err := client.GetMonetaryBankAccounts()
	if err != nil {
		panic(err)
	}
	fmt.Println(json.Marshal(bankAccounts))
}
