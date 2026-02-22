package parser

import (
	"testing"
)

func TestHSBCParser_Parse(t *testing.T) {
	p := &HSBCParser{}

	pages := []string{
		`HSBC UK Bank plc
Your Statement
Account name: Jane Doe
Sort code: 40-12-34
Account number: 87654321

Date Payment type and details Paid out Paid in Balance
15 Jan 24 CARD PAYMENT TO TESCO STORES £25.99 £1,234.56
16 Jan 24 DIRECT DEBIT SKY UK LIMITED £45.00 £1,189.56
17 Jan 24 CREDIT SALARY EMPLOYER LTD £2,500.00 £3,689.56
18 Jan 24 ATM WITHDRAWAL LONDON £100.00 £3,589.56`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.AccountNumber != "87654321" {
		t.Errorf("account number: got %q, want %q", info.AccountNumber, "87654321")
	}

	if info.SortCode != "40-12-34" {
		t.Errorf("sort code: got %q, want %q", info.SortCode, "40-12-34")
	}

	if len(info.Transactions) == 0 {
		t.Fatal("expected transactions, got none")
	}

	// Verify at least some transactions were parsed
	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %s | %s | %.2f | %.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}
}

func TestHSBCParser_SlashDates(t *testing.T) {
	p := &HSBCParser{}

	// Some HSBC variants use slash dates
	pages := []string{
		`HSBC
Date Details Paid out Paid in Balance
15/01/2024 CARD PAYMENT TESCO 25.99 1,234.56`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) == 0 {
		t.Fatal("expected at least one transaction")
	}
}
