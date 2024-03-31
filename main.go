package main

import (
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

	for _, bankAccount := range bankAccounts {
		lastId := 0
		for {
			payments, err := client.GetPayments(bankAccount.Id, lastId)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if len(payments) == 0 {
				break
			}

			fmt.Println(payments)
			// TODO: Create payment in firefly for every bunq payment until the first payment that is already in firefly

			lastId = payments[len(payments)-1].Id
		}
	}
}
