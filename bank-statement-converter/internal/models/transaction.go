package models

// Transaction represents a single bank statement transaction.
type Transaction struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // DEBIT or CREDIT
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
	ParseMethod string  `json:"parseMethod,omitempty"` // debug: which parser method matched
}

// BankType represents supported bank statement formats.
type BankType string

const (
	BankMetro    BankType = "metro"
	BankHSBC     BankType = "hsbc"
	BankBarclays BankType = "barclays"
)

// DebugLine captures what the parser did with each input line.
type DebugLine struct {
	LineNum  int    `json:"lineNum"`
	Text     string `json:"text"`
	HasDate  bool   `json:"hasDate"`
	HasTab   bool   `json:"hasTab"`
	Result   string `json:"result"` // "parsed", "skipped", "continuation", "header"
	Method   string `json:"method,omitempty"`
	TabParts int    `json:"tabParts,omitempty"`
}

// StatementInfo holds metadata extracted from the statement.
type StatementInfo struct {
	Bank            BankType
	AccountHolder   string
	AccountNumber   string
	SortCode        string
	StatementPeriod string
	Transactions    []Transaction
	DebugLines      []DebugLine
}
