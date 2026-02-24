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

	// Check first transaction (debit)
	txn := info.Transactions[0]
	if txn.Date != "15/01/2024" {
		t.Errorf("txn[0].Date: got %q, want %q", txn.Date, "15/01/2024")
	}
	if txn.Amount != 25.99 {
		t.Errorf("txn[0].Amount: got %f, want %f", txn.Amount, 25.99)
	}
	if txn.Type != "DEBIT" {
		t.Errorf("txn[0].Type: got %q, want %q", txn.Type, "DEBIT")
	}

	// Check second transaction (debit)
	txn = info.Transactions[1]
	if txn.Amount != 45.00 {
		t.Errorf("txn[1].Amount: got %f, want %f", txn.Amount, 45.00)
	}
	if txn.Type != "DEBIT" {
		t.Errorf("txn[1].Type: got %q, want %q", txn.Type, "DEBIT")
	}

	// Check salary credit — this is the key test for the Money In bug
	txn = info.Transactions[2]
	if txn.Date != "17/01/2024" {
		t.Errorf("txn[2].Date: got %q, want %q", txn.Date, "17/01/2024")
	}
	if txn.Amount != 2500.00 {
		t.Errorf("txn[2].Amount: got %f, want %f", txn.Amount, 2500.00)
	}
	if txn.Type != "CREDIT" {
		t.Errorf("txn[2].Type: got %q, want %q (Money In incorrectly classified)", txn.Type, "CREDIT")
	}

	// Check fourth transaction (debit after credit)
	txn = info.Transactions[3]
	if txn.Amount != 15.49 {
		t.Errorf("txn[3].Amount: got %f, want %f", txn.Amount, 15.49)
	}
	if txn.Type != "DEBIT" {
		t.Errorf("txn[3].Type: got %q, want %q", txn.Type, "DEBIT")
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

	if len(info.Transactions) == 0 {
		t.Fatal("expected at least one transaction")
	}
}

func TestMetroBankParser_CreditDetection(t *testing.T) {
	p := &MetroBankParser{}

	// Scenario: Multiple credits and debits with balance progression
	// Opening balance: 1,000.00
	// Debit 50.00 → 950.00
	// Credit 200.00 → 1,150.00
	// Debit 30.00 → 1,120.00
	// Credit 500.00 → 1,620.00
	pages := []string{
		`Date Description Money out Money in Balance
Opening balance 1,000.00
01/02/2024 CARD PAYMENT SHOP 50.00 950.00
02/02/2024 FASTER PAYMENT RECEIVED J DOE 200.00 1,150.00
03/02/2024 DIRECT DEBIT NETFLIX 30.00 1,120.00
04/02/2024 BANK CREDIT REFUND 500.00 1,620.00`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) != 4 {
		t.Fatalf("transactions: got %d, want 4", len(info.Transactions))
	}

	tests := []struct {
		idx      int
		amount   float64
		typ      string
		balance  float64
	}{
		{0, 50.00, "DEBIT", 950.00},
		{1, 200.00, "CREDIT", 1150.00},
		{2, 30.00, "DEBIT", 1120.00},
		{3, 500.00, "CREDIT", 1620.00},
	}

	for _, tt := range tests {
		txn := info.Transactions[tt.idx]
		if txn.Amount != tt.amount {
			t.Errorf("txn[%d].Amount: got %f, want %f", tt.idx, txn.Amount, tt.amount)
		}
		if txn.Type != tt.typ {
			t.Errorf("txn[%d].Type: got %q, want %q", tt.idx, txn.Type, tt.typ)
		}
		if txn.Balance != tt.balance {
			t.Errorf("txn[%d].Balance: got %f, want %f", tt.idx, txn.Balance, tt.balance)
		}
	}
}

func TestMetroBankParser_OpeningBalance(t *testing.T) {
	p := &MetroBankParser{}

	// First transaction is a credit — opening balance is needed to classify it
	pages := []string{
		`Date Description Paid out Paid in Balance
Opening balance 5,000.00
01/01/2024 BANK CREDIT SALARY 2,500.00 7,500.00`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) != 1 {
		t.Fatalf("transactions: got %d, want 1", len(info.Transactions))
	}

	txn := info.Transactions[0]
	if txn.Type != "CREDIT" {
		t.Errorf("txn[0].Type: got %q, want %q", txn.Type, "CREDIT")
	}
	if txn.Amount != 2500.00 {
		t.Errorf("txn[0].Amount: got %f, want %f", txn.Amount, 2500.00)
	}
}

func TestMetroBankParser_MoneyInMoneyOut(t *testing.T) {
	p := &MetroBankParser{}

	// Metro Bank sometimes uses "Money in" / "Money out" column headers
	pages := []string{
		`Metro Bank Statement

Date Transaction details Money out Money in Balance
Balance brought forward 2,000.00
10/01/2024 CARD PAYMENT ASDA 35.50 1,964.50
11/01/2024 FASTER PAYMENT RECEIVED 1,000.00 2,964.50
12/01/2024 STANDING ORDER RENT 750.00 2,214.50
13/01/2024 INTEREST PAYMENT 5.25 2,219.75`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) != 4 {
		t.Fatalf("transactions: got %d, want 4", len(info.Transactions))
	}

	tests := []struct {
		idx    int
		amount float64
		typ    string
	}{
		{0, 35.50, "DEBIT"},
		{1, 1000.00, "CREDIT"},
		{2, 750.00, "DEBIT"},
		{3, 5.25, "CREDIT"},
	}

	for _, tt := range tests {
		txn := info.Transactions[tt.idx]
		if txn.Amount != tt.amount {
			t.Errorf("txn[%d].Amount: got %f, want %f", tt.idx, txn.Amount, tt.amount)
		}
		if txn.Type != tt.typ {
			t.Errorf("txn[%d].Type: got %q, want %q (Money In/Out classification)", tt.idx, txn.Type, tt.typ)
		}
	}
}

func TestClassifyByBalance(t *testing.T) {
	tests := []struct {
		name    string
		amt     float64
		bal     float64
		prevBal float64
		desc    string
		want    string
	}{
		{"debit with balance", 50.00, 950.00, 1000.00, "CARD PAYMENT", "DEBIT"},
		{"credit with balance", 200.00, 1200.00, 1000.00, "SALARY", "CREDIT"},
		{"no prev balance, debit desc", 50.00, 950.00, 0, "CARD PAYMENT TESCO", "DEBIT"},
		{"no prev balance, credit desc", 200.00, 1200.00, 0, "SALARY", "CREDIT"},
		{"no prev balance, transfer in", 500.00, 1500.00, 0, "TRANSFER IN FROM SAVINGS", "CREDIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyByBalance(tt.amt, tt.bal, tt.prevBal, tt.desc)
			if got != tt.want {
				t.Errorf("classifyByBalance(%f, %f, %f, %q) = %q, want %q",
					tt.amt, tt.bal, tt.prevBal, tt.desc, got, tt.want)
			}
		})
	}
}
