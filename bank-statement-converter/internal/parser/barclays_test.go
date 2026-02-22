package parser

import (
	"testing"
)

func TestBarclaysParser_Parse(t *testing.T) {
	p := &BarclaysParser{}

	pages := []string{
		`Barclays Bank UK PLC
Your Statement
Sort code: 20-00-00
Account number: 11223344

Date Description Money out Money in Balance
15/01/2024 CARD PAYMENT TESCO STORES 25.99 1,234.56
16/01/2024 DIRECT DEBIT SKY UK 45.00 1,189.56
17/01/2024 BGC SALARY EMPLOYER 2,500.00 3,689.56
18/01/2024 CARD PAYMENT AMAZON 15.49 3,674.07`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.AccountNumber != "11223344" {
		t.Errorf("account number: got %q, want %q", info.AccountNumber, "11223344")
	}

	if info.SortCode != "20-00-00" {
		t.Errorf("sort code: got %q, want %q", info.SortCode, "20-00-00")
	}

	if len(info.Transactions) == 0 {
		t.Fatal("expected transactions, got none")
	}

	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %s | %s | %.2f | %.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}
}

func TestBarclaysParser_TextDates(t *testing.T) {
	p := &BarclaysParser{}

	pages := []string{
		`Barclays
Date Description Money out Money in Balance
15 Jan 2024 CARD PAYMENT TESCO 25.99 1,234.56`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) == 0 {
		t.Fatal("expected at least one transaction")
	}
}
