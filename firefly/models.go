package firefly

import (
	"strings"
	"time"
)

// FIREFLY COMMON MODELS

type ResponseMeta struct {
	Pagination *Pagination `json:"pagination"`
}

type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

// FIREFLY ACCOUNT MODELS

type AccountField string

const (
	AllFields   AccountField = "all"
	IbanField   AccountField = "iban"
	NameField   AccountField = "name"
	NumberField AccountField = "number"
	IdField     AccountField = "id"
)

type AccountType string

const (
	AllTypes           AccountType = "all"
	AssetType          AccountType = "asset"
	CashType           AccountType = "cash"
	ExpenseType        AccountType = "expense"
	RevenueType        AccountType = "revenue"
	SpecialType        AccountType = "special"
	HiddenType         AccountType = "hidden"
	LiabilityType      AccountType = "liability"
	LiabilitiesType    AccountType = "liabilities"
	ImportType         AccountType = "import"
	InitialBalanceType AccountType = "initial-balance"
	Reconciliation     AccountType = "reconciliation"
)

type AccountRole string

const (
	DefaultAsset    AccountRole = "defaultAsset"
	SharedAsset     AccountRole = "sharedAsset"
	SavingAsset     AccountRole = "savingAsset"
	CcAsset         AccountRole = "ccAsset"
	CashWalletAsset AccountRole = "cashWalletAsset"
)

type Account struct {
	CreatedAt             *time.Time  `json:"created_at"`
	UpdatedAt             *time.Time  `json:"updated_at"`
	Active                bool        `json:"active"`
	Order                 int         `json:"order"`
	Name                  string      `json:"name"`
	Type                  AccountType `json:"type"`
	AccountRole           AccountRole `json:"account_role"`
	CurrencyId            string      `json:"currency_id"`
	CurrencyCode          string      `json:"currency_code"`
	CurrencySymbol        string      `json:"currency_symbol"`
	CurrencyDecimalPlaces int         `json:"currency_decimal_places"`
	CurrentBalance        string      `json:"current_balance"`
	CurrentBalanceDate    *time.Time  `json:"current_balance_date"`
	Iban                  string      `json:"iban"`
	Bic                   string      `json:"bic"`
	AccountNumber         string      `json:"account_number"`
	OpeningBalance        string      `json:"opening_balance"`
	CurrentDept           string      `json:"current_dept"`
	OpeningBalanceDate    *time.Time  `json:"opening_balance_date"`
	VirtualBalance        string      `json:"virtual_balance"`
	IncludeNetWorth       bool        `json:"include_net_worth"`
	CreditCardType        string      `json:"credit_card_type"`
	MonthlyPaymentDate    *time.Time  `json:"monthly_payment_date"`
	LiabilityType         string      `json:"liability_type"`
	LiabilityDirection    string      `json:"liability_direction"`
	Interest              string      `json:"interest"`
	InterestPeriod        string      `json:"interest_period"`
	Notes                 string      `json:"notes"`
	Latitude              float32     `json:"latitude"`
	Longitude             float32     `json:"longitude"`
	ZoomLevel             int         `json:"zoom_level"`
}

type AccountRead struct {
	Type       string   `json:"type"`
	Id         string   `json:"id"`
	Attributes *Account `json:"attributes"`
}

type AccountsResponse struct {
	Data []*AccountRead `json:"data"`
	Meta *ResponseMeta  `json:"meta"`
}

type AccountResponse struct {
	Data *AccountRead `json:"data"`
}

type AccountRequest struct {
	Name               string      `json:"name"`
	Type               AccountType `json:"type"`
	Iban               string      `json:"iban"`
	Bic                string      `json:"bic"`
	AccountNumber      string      `json:"account_number"`
	OpeningBalance     string      `json:"opening_balance"`
	OpeningBalanceDate *time.Time  `json:"opening_balance_date"`
	AccountRole        AccountRole `json:"account_role"`
	Notes              string      `json:"notes"`
}

// FIREFLY TRANSACTION MODELS

type TransactionSearchQuery struct {
	ExternalIdIs string
	AccountNrIs  string
}

func (q *TransactionSearchQuery) Encode() string {
	result := ""

	if q.ExternalIdIs != "" {
		result += " external_id_is:" + q.ExternalIdIs
	}

	if q.ExternalIdIs != "" {
		result += " account_nr_is:" + q.AccountNrIs
	}

	return strings.Trim(result, " ")
}

type TransactionType string

const (
	WithdrawalTransaction TransactionType = "withdrawal"
	DepositTransaction    TransactionType = "deposit"
	TransferTransaction   TransactionType = "transfer"
)

type TransactionSplit struct {
	User                 string          `json:"user"`
	TransactionJournalId string          `json:"transaction_journal_id"`
	Type                 TransactionType `json:"type"`
	Date                 *time.Time      `json:"date"`
	Amount               string          `json:"amount"`
	Description          string          `json:"description"`
	SourceId             string          `json:"source_id"`
	SourceName           string          `json:"source_name"`
	SourceIban           string          `json:"source_iban"`
	DestinationId        string          `json:"destionation_id"`
	DestinationName      string          `json:"destination_name"`
	DestinationIban      string          `json:"destination_iban"`
	Notes                string          `json:"notes"`
	ExternalId           string          `json:"external_id"`
}

type Transaction struct {
	CreatedAt    *time.Time          `json:"created_at"`
	UpdatedAt    *time.Time          `json:"updated_at"`
	User         string              `json:"user"`
	GroupTitle   string              `json:"group_title"`
	Transactions []*TransactionSplit `json:"transactions"`
}

type TransactionRead struct {
	Type       string       `json:"type"`
	Id         string       `json:"id"`
	Attributes *Transaction `json:"attributes"`
}

type TransactionsResponse struct {
	Data []*TransactionRead `json:"data"`
	Meta *ResponseMeta      `json:"meta"`
}

type TransactionResponse struct {
	Data *TransactionRead `json:"data"`
}

type TransactionSplitRequest struct {
	Type          TransactionType `json:"type"`
	Date          *time.Time      `json:"date"`
	Amount        string          `json:"amount"`
	Description   string          `json:"description"`
	CurrencyCode  string          `json:"currency_code"`
	SourceId      string          `json:"source_id"`
	DestinationId string          `json:"destination_id"`
	Notes         string          `json:"notes"`
	ExternalId    string          `json:"external_id"`
}

type TransactionRequest struct {
	Transactions         []*TransactionSplitRequest `json:"transactions"`
	ErrorIfDuplicateHash bool                       `json:"error_if_duplicate_hash"`
}
