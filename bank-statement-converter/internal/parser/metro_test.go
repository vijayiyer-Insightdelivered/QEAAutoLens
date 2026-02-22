package parser

import (
	"testing"
)

func TestMetroBankParser_Parse(t *testing.T) {
	p := &MetroBankParser{}

	pages := []string{
		`Metro Bank
Account Statement
Account holder: John Smith
Sort code: 23-05-80
Account number: 12345678
Statement period: 01/01/2024 to 31/01/2024

Date Description Paid out Paid in Balance
15/01/2024 CARD PAYMENT TESCO STORES 25.99 1,234.56
16/01/2024 DIRECT DEBIT SKY UK LTD 45.00 1,189.56
17/01/2024 BANK CREDIT SALARY 2,500.00 3,689.56
18/01/2024 CARD PAYMENT AMAZON UK 15.49 3,674.07`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.AccountNumber != "12345678" {
		t.Errorf("account number: got %q, want %q", info.AccountNumber, "12345678")
	}

	if info.SortCode != "23-05-80" {
		t.Errorf("sort code: got %q, want %q", info.SortCode, "23-05-80")
	}

	if len(info.Transactions) != 4 {
		t.Fatalf("transactions: got %d, want 4", len(info.Transactions))
	}

	// Check first transaction
	txn := info.Transactions[0]
	if txn.Date != "15/01/2024" {
		t.Errorf("txn[0].Date: got %q, want %q", txn.Date, "15/01/2024")
	}
	if txn.Amount != 25.99 {
		t.Errorf("txn[0].Amount: got %f, want %f", txn.Amount, 25.99)
	}

	// Check salary credit
	txn = info.Transactions[2]
	if txn.Date != "17/01/2024" {
		t.Errorf("txn[2].Date: got %q, want %q", txn.Date, "17/01/2024")
	}
	if txn.Amount != 2500.00 {
		t.Errorf("txn[2].Amount: got %f, want %f", txn.Amount, 2500.00)
	}
}

func TestMetroBankParser_FullPattern(t *testing.T) {
	p := &MetroBankParser{}

	// Test with explicit paid out / paid in / balance columns
	pages := []string{
		`Date Description Paid out Paid in Balance
15/01/2024 CARD PAYMENT TESCO 25.99 1,234.56
16/01/2024 TRANSFER IN 500.00 1,734.56`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These should be parsed with the simple pattern (date + desc + amount)
	if len(info.Transactions) == 0 {
		t.Fatal("expected at least one transaction")
	}
}
