package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/daanvanberkel/fireflyiiibunq/bunq"
	"github.com/daanvanberkel/fireflyiiibunq/firefly"
	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.Level = logrus.InfoLevel
	log.Out = os.Stdout

	arguments := os.Args
	var date time.Time
	if len(arguments) >= 2 {
		var err error
		date, err = time.Parse("2006-01-02", arguments[1])
		if err != nil {
			panic(err)
		}
	} else {
		date = time.Now()
	}
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	log.WithField("date", date.Format("2006-01-02")).Info("Starting bunq -> firefly sync")

	config, err := util.LoadConfig()
	if err != nil {
		panic(err)
	}

	fireflyClient, err := firefly.NewFireflyClient(config, log)
	if err != nil {
		panic(err)
	}

	bunqClient, err := bunq.NewBunqClient(config, log)
	if err != nil {
		panic(err)
	}

	bankAccounts, err := bunqClient.GetMonetaryBankAccounts()
	if err != nil {
		panic(err)
	}

	for _, bankAccount := range bankAccounts {
		iban, err := bankAccount.GetIBAN()
		if err != nil {
			log.WithError(err).WithField("bankAccount", bankAccount).Error("Cannot get IBAN for bankaccount")
			continue
		}

		assetAccount, err := fireflyClient.FindOrCreateAssetAccount(iban, &firefly.AccountRequest{
			Name:        bankAccount.DisplayName + " - " + bankAccount.Description, // Adding description after display name to prevent naming collisions
			Type:        firefly.AssetType,
			Iban:        iban,
			AccountRole: firefly.DefaultAsset,
			Notes:       "Created by Bunq sync on" + time.Now().String(),
		})
		if err != nil {
			log.WithError(err).WithField("iban", iban).Error("Cannot find or create firefly account")
			continue
		}

		lastId := 0
		processTransactions := true
		for processTransactions {
			payments, err := bunqClient.GetPayments(bankAccount.Id, lastId)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if len(payments) == 0 {
				// No payments found, stop loop
				processTransactions = false
				break
			}

			for _, payment := range payments {
				paymentLogger := log.WithFields(logrus.Fields{
					"paymentId":  payment.Id,
					"sourceIban": payment.Alias.Iban,
					"targetIban": payment.CounterpartyAlias.Iban,
					"date":       payment.Created,
				})
				isWithdrawal := payment.Amount.Value[0] == '-'

				if date.Compare(payment.Created.Time) >= 1 {
					processTransactions = false
					paymentLogger.Info("Received payment too far in the past, stop processing")
					continue
				}

				paymentLogger.Info("Start processing payment")

				transactions, err := fireflyClient.SearchTransactions(&firefly.TransactionSearchQuery{
					ExternalIdIs: strconv.Itoa(payment.Id),
					AccountNrIs:  iban,
				}, 1)
				if err != nil {
					paymentLogger.WithError(err).Error("Error while fetching transaction from firefly")
					continue
				}

				if transactions.Meta.Pagination.Total > 0 {
					// Transaction already in Firefly, stop processing
					paymentLogger.Info("Payment already in firefly, skipping payment")
					continue
				}

				counterPartyAssetAccount, err := findCounterPartyAssetAccount(fireflyClient, payment, paymentLogger)
				if err != nil {
					paymentLogger.WithError(err).Error("Failed searching for counterparty asset account, skipping payment")
					continue
				}

				if counterPartyAssetAccount != nil {
					// Make a transfer between two asset accounts
					if err := createTransactionSplitForPayment(firefly.TransferTransaction, payment, assetAccount.Id, counterPartyAssetAccount.Id, fireflyClient, true); err != nil {
						paymentLogger.WithError(err).Error("Cannot create new transaction in firefly")
						continue
					}

					paymentLogger.Info("Created new transaction in firefly")
					continue
				}

				// Make a normal withdrawal or deposit
				var accountType firefly.AccountType
				if isWithdrawal {
					accountType = firefly.ExpenseType
				} else {
					accountType = firefly.RevenueType
				}

				account, err := findOrCreateAccountForPayment(fireflyClient, payment, accountType, paymentLogger)
				if err != nil {
					paymentLogger.WithError(err).Error("Cannot search for expense or revenue accounts by iban in firefly")
					continue
				}

				var transactionType firefly.TransactionType
				if isWithdrawal {
					transactionType = firefly.WithdrawalTransaction
				} else {
					transactionType = firefly.DepositTransaction
				}

				if err := createTransactionSplitForPayment(transactionType, payment, assetAccount.Id, account.Id, fireflyClient, false); err != nil {
					paymentLogger.WithError(err).Error("Cannot create new transaction in firefly")
					continue
				}

				paymentLogger.Info("Created new transaction in firefly")
			}

			lastId = payments[len(payments)-1].Id
		}
	}
}

func findCounterPartyAssetAccount(fireflyClient *firefly.FireflyClient, payment *bunq.BunqPayment, log *logrus.Entry) (*firefly.AccountRead, error) {
	if payment.CounterpartyAlias.Iban == "" {
		return nil, nil
	}

	counterpartyAssetAccounts, err := fireflyClient.SearchAccounts(payment.CounterpartyAlias.Iban, firefly.IbanField, firefly.AssetType, 1)
	if err != nil {
		log.WithError(err).Error("Failed searching for counterparty asset account, skipping payment")
		return nil, err
	}

	if counterpartyAssetAccounts.Meta.Pagination.Total == 0 {
		return nil, nil
	}

	return counterpartyAssetAccounts.Data[0], nil
}

func findOrCreateAccountForPayment(fireflyClient *firefly.FireflyClient, payment *bunq.BunqPayment, accountType firefly.AccountType, log *logrus.Entry) (*firefly.AccountRead, error) {
	if payment.CounterpartyAlias.Iban != "" {
		// Find accounts by iban
		accounts, err := fireflyClient.SearchAccounts(payment.CounterpartyAlias.Iban, firefly.IbanField, accountType, 1)
		if err != nil {
			log.WithError(err).Error("Cannot search for expense or revenue accounts by iban in firefly")
			return nil, err
		}

		if accounts.Meta.Pagination.Total > 0 {
			return accounts.Data[0], nil
		}
	}

	if payment.CounterpartyAlias.DisplayName != "" {
		// Find accounts by name
		accounts, err := fireflyClient.SearchAccounts(payment.CounterpartyAlias.DisplayName, firefly.NameField, accountType, 1)
		if err != nil {
			log.WithError(err).Error("Cannot search for expense or revenue accounts by name in firefly")
			return nil, err
		}

		if accounts.Meta.Pagination.Total > 0 {
			return accounts.Data[0], nil
		}
	}

	// Create new account in firefly
	accountRequest := &firefly.AccountRequest{
		Name:  payment.CounterpartyAlias.DisplayName,
		Type:  accountType,
		Iban:  payment.CounterpartyAlias.Iban,
		Notes: "Created by Bunq sync on " + time.Now().String(),
	}
	return fireflyClient.CreateAccount(accountRequest)
}

func createTransactionSplitForPayment(transactionType firefly.TransactionType, payment *bunq.BunqPayment, sourceId string, destinationId string, fireflyClient *firefly.FireflyClient, errorIfDuplicateHash bool) error {
	var description string
	if payment.Description == "" {
		description = "(empty)"
	} else {
		description = payment.Description
	}

	isWithdrawal := payment.Amount.Value[0] == '-'

	oldSourceId := sourceId
	if !isWithdrawal {
		sourceId = destinationId
		destinationId = oldSourceId
	}

	transaction := &firefly.TransactionSplitRequest{
		Type:          transactionType,
		Date:          &payment.Created.Time,
		Amount:        strings.Trim(payment.Amount.Value, "-"),
		Description:   description,
		CurrencyCode:  payment.Amount.Currency,
		SourceId:      sourceId,
		DestinationId: destinationId,
		Notes:         "Created by Bunq sync on " + time.Now().String(),
		ExternalId:    strconv.Itoa(payment.Id),
	}
	_, err := fireflyClient.CreateTransaction(&firefly.TransactionRequest{
		Transactions:         []*firefly.TransactionSplitRequest{transaction},
		ErrorIfDuplicateHash: errorIfDuplicateHash,
	})

	return err
}
