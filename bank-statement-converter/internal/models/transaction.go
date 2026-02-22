package models

// Transaction represents a single bank statement transaction.
type Transaction struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // DEBIT or CREDIT
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
}

// BankType represents supported bank statement formats.
type BankType string

const (
	BankMetro    BankType = "metro"
	BankHSBC     BankType = "hsbc"
	BankBarclays BankType = "barclays"
)

// StatementInfo holds metadata extracted from the statement.
type StatementInfo struct {
	Bank            BankType
	AccountHolder   string
	AccountNumber   string
	SortCode        string
	StatementPeriod string
	Transactions    []Transaction
}
