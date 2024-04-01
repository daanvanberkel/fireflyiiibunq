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
	log.Level = logrus.DebugLevel
	log.Out = os.Stdout

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
			Name:        bankAccount.DisplayName,
			Type:        firefly.AssetType,
			Iban:        iban,
			AccountRole: firefly.DefaultAsset,
			Notes:       "Created by Bunq sync on" + time.Now().String(),
		})
		if err != nil {
			log.WithError(err).WithField("iban", iban).Error("Cannot find or create firefly account")
			continue
		}
		fmt.Println(assetAccount)

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
				// TODO: Only check payments for current day (or given day using cli arguments)

				paymentLogger := log.WithFields(logrus.Fields{
					"paymentId":  payment.Id,
					"sourceIban": payment.Alias.Iban,
					"targetIban": payment.CounterpartyAlias.Iban,
					"date":       payment.Created,
				})
				isWithdrawal := payment.Amount.Value[0] == '-'

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

				counterpartyAssetAccounts, err := fireflyClient.SearchAccounts(payment.CounterpartyAlias.Iban, firefly.IbanField, firefly.AssetType, 1)
				if err != nil {
					paymentLogger.WithError(err).Error("Failed searching for counterparty asset account, skipping payment")
					continue
				}

				if counterpartyAssetAccounts.Meta.Pagination.Total > 0 {
					// Make a transfer between two asset accounts
					counterpartyAssetAccount := counterpartyAssetAccounts.Data[0]

					var sourceId string
					if isWithdrawal {
						sourceId = assetAccount.Id
					} else {
						sourceId = counterpartyAssetAccount.Id
					}

					var destinationId string
					if isWithdrawal {
						destinationId = counterpartyAssetAccount.Id
					} else {
						destinationId = assetAccount.Id
					}

					transaction := &firefly.TransactionSplitRequest{
						Type:          firefly.TransferTransaction,
						Date:          &payment.Created.Time,
						Amount:        strings.Trim(payment.Amount.Value, "-"),
						Description:   payment.Description,
						CurrencyCode:  payment.Amount.Currency,
						SourceId:      sourceId,
						DestinationId: destinationId,
						Notes:         "Created by Bunq sync on " + time.Now().String(),
						ExternalId:    strconv.Itoa(payment.Id),
					}

					_, err := fireflyClient.CreateTransaction(&firefly.TransactionRequest{
						Transactions: []*firefly.TransactionSplitRequest{transaction},
					})
					if err != nil {
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

				accounts, err := fireflyClient.SearchAccounts(payment.CounterpartyAlias.Iban, firefly.IbanField, accountType, 1)
				if err != nil {
					paymentLogger.WithError(err).Error("Cannot search for expense or revenue accounts in firefly")
					continue
				}

				var account *firefly.AccountRead
				if accounts.Meta.Pagination.Total > 0 {
					account = accounts.Data[0]
				} else {
					// Create new account in firefly
					accountRequest := &firefly.AccountRequest{
						Name:  payment.CounterpartyAlias.DisplayName,
						Type:  accountType,
						Iban:  payment.CounterpartyAlias.Iban,
						Notes: "Created by Bunq sync on " + time.Now().String(),
					}
					account, err = fireflyClient.CreateAccount(accountRequest)
					if err != nil {
						paymentLogger.WithError(err).WithField("request", accountRequest).Error("Cannot create new account in firefly")
						continue
					}
				}

				var transactionType firefly.TransactionType
				if isWithdrawal {
					transactionType = firefly.WithdrawalTransaction
				} else {
					transactionType = firefly.DepositTransaction
				}

				var sourceId string
				if isWithdrawal {
					sourceId = assetAccount.Id
				} else {
					sourceId = account.Id
				}

				var destinationId string
				if isWithdrawal {
					destinationId = account.Id
				} else {
					destinationId = assetAccount.Id
				}

				transaction := &firefly.TransactionSplitRequest{
					Type:          transactionType,
					Date:          &payment.Created.Time,
					Amount:        strings.Trim(payment.Amount.Value, "-"),
					Description:   payment.Description,
					CurrencyCode:  payment.Amount.Currency,
					SourceId:      sourceId,
					DestinationId: destinationId,
					Notes:         "Created by Bunq sync on " + time.Now().String(),
					ExternalId:    strconv.Itoa(payment.Id),
				}
				_, err = fireflyClient.CreateTransaction(&firefly.TransactionRequest{
					Transactions: []*firefly.TransactionSplitRequest{transaction},
				})
				if err != nil {
					paymentLogger.WithError(err).WithField("request", transaction).Error("Cannot create new transaction in firefly")
					continue
				}

				paymentLogger.Info("Created new transaction in firefly")
			}

			lastId = payments[len(payments)-1].Id
		}
	}
}
