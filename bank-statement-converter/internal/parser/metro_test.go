package parser

import (
	"strings"
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

func TestMetroBankParser_TextDateFormat(t *testing.T) {
	p := &MetroBankParser{}

	// Real-world Metro Bank business statement format:
	// - Text dates (DD MMM YYYY) instead of DD/MM/YYYY
	// - "Money out (£)" / "Money in (£)" column headers
	// - Multi-line descriptions (description continues on next line)
	// - "ACCOUNT NAME:" label in uppercase
	pages := []string{
		`Business Bank Account Statement
BIC: MYMBGB2L IBAN: GB45MYMB23058056354379

ACCOUNT NAME: AURORA CAR SALES LTD

From: 01 SEP 2025 To: 30 SEP 2025 Account number 56354379
Opening balance £7,225.15 Sort code 23-05-80

Date Transaction Money out (£) Money in (£) Balance (£)

Balance brought forward 7,225.15

01 SEP 2025 Inward Payment 12,495.00 19,720.15
sd vehicles

02 SEP 2025 Outward Faster Payment SD VEHICLES 1.00 19,719.15
NA

02 SEP 2025 Outward Faster Payment SD VEHICLES 8,435.00 11,284.15
NA

04 SEP 2025 Outward Faster Payment MR Benjamin Hamer 1.00 11,283.15
NA

04 SEP 2025 Outward Faster Payment MR Benjamin Hamer 10,498.00 785.15
NA`,
		`Statement number 4
Business Bank Account number 56354379
Sort code 23-05-80

Date Transaction Money out (£) Money in (£) Balance (£)

05 SEP 2025 Inward Payment 15,995.00 16,780.15
sd vehicles

05 SEP 2025 Outward Faster Payment McMillan Alloys Ltd 744.00 16,036.15
NA

26 SEP 2025 Internet Banking Chgs 5.00 16,031.15

26 SEP 2025 Transaction Charges 6.30 16,024.85

26 SEP 2025 Account Maintenance Fee 8.00 16,016.85`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Metadata checks
	if info.AccountNumber != "56354379" {
		t.Errorf("account number: got %q, want %q", info.AccountNumber, "56354379")
	}
	if info.SortCode != "23-05-80" {
		t.Errorf("sort code: got %q, want %q", info.SortCode, "23-05-80")
	}
	if info.AccountHolder != "AURORA CAR SALES LTD" {
		t.Errorf("account holder: got %q, want %q", info.AccountHolder, "AURORA CAR SALES LTD")
	}

	// Should parse all 10 transactions across both pages
	// Page 1: 5 txns (01 SEP inward, 02 SEP outward x2, 04 SEP outward x2)
	// Page 2: 5 txns (05 SEP inward, 05 SEP outward, 26 SEP x3 charges)
	if len(info.Transactions) != 10 {
		t.Fatalf("transactions: got %d, want 10; parsed transactions: %+v", len(info.Transactions), info.Transactions)
	}

	// Verify first transaction: Inward Payment (credit)
	txn := info.Transactions[0]
	if txn.Amount != 12495.00 {
		t.Errorf("txn[0].Amount: got %f, want %f", txn.Amount, 12495.00)
	}
	if txn.Type != "CREDIT" {
		t.Errorf("txn[0].Type: got %q, want %q", txn.Type, "CREDIT")
	}
	if txn.Balance != 19720.15 {
		t.Errorf("txn[0].Balance: got %f, want %f", txn.Balance, 19720.15)
	}

	// Verify second transaction: Outward Faster Payment (debit)
	txn = info.Transactions[1]
	if txn.Amount != 1.00 {
		t.Errorf("txn[1].Amount: got %f, want %f", txn.Amount, 1.00)
	}
	if txn.Type != "DEBIT" {
		t.Errorf("txn[1].Type: got %q, want %q", txn.Type, "DEBIT")
	}
	if txn.Balance != 19719.15 {
		t.Errorf("txn[1].Balance: got %f, want %f", txn.Balance, 19719.15)
	}

	// Verify page 2 first transaction: Inward Payment (credit)
	txn = info.Transactions[5]
	if txn.Amount != 15995.00 {
		t.Errorf("txn[5].Amount: got %f, want %f", txn.Amount, 15995.00)
	}
	if txn.Type != "CREDIT" {
		t.Errorf("txn[5].Type: got %q, want %q", txn.Type, "CREDIT")
	}

	// Verify charges appear as transactions (Internet Banking Chgs)
	txn = info.Transactions[7]
	if txn.Amount != 5.00 {
		t.Errorf("txn[7].Amount: got %f, want %f", txn.Amount, 5.00)
	}
	if txn.Type != "DEBIT" {
		t.Errorf("txn[7].Type: got %q, want %q", txn.Type, "DEBIT")
	}
}

func TestMetroBankParser_TextDateUppercaseMonth(t *testing.T) {
	p := &MetroBankParser{}

	// Verify uppercase month names (SEP, OCT, etc.) are handled
	pages := []string{
		`Date Transaction Money out Money in Balance
Balance brought forward 1,000.00
01 SEP 2025 CARD PURCHASE TESCO 50.00 950.00
15 OCT 2025 FASTER PAYMENT RECEIVED 500.00 1,450.00
03 NOV 2025 DIRECT DEBIT SKY 30.00 1,420.00`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) != 3 {
		t.Fatalf("transactions: got %d, want 3", len(info.Transactions))
	}

	tests := []struct {
		idx     int
		date    string
		amount  float64
		typ     string
		balance float64
	}{
		{0, "01 SEP 2025", 50.00, "DEBIT", 950.00},
		{1, "15 OCT 2025", 500.00, "CREDIT", 1450.00},
		{2, "03 NOV 2025", 30.00, "DEBIT", 1420.00},
	}

	for _, tt := range tests {
		txn := info.Transactions[tt.idx]
		if txn.Date != tt.date {
			t.Errorf("txn[%d].Date: got %q, want %q", tt.idx, txn.Date, tt.date)
		}
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

func TestMetroBankParser_ColumnSeparatedFormat(t *testing.T) {
	p := &MetroBankParser{}

	// Simulates a page where PDF extraction outputs descriptions and amounts
	// in separate blocks (column-separated format). This is based on real
	// Metro Bank business statement OCR output.
	//
	// Balance progression (opening = 10,000.00):
	//   +5,000 = 15,000.00 (credit: Inward Payment)
	//   -1,000 = 14,000.00 (debit: Outward Faster Payment)
	//   -500   = 13,500.00 (debit: Direct Debit)
	//   +2,000 = 15,500.00 (credit: Inward Payment)
	pages := []string{
		`Statement number 4
Business Bank Account number 56354379
Sort code 23-05-80

Date Transaction

05 SEP 2025 Inward Payment
sd vehicles
05 SEP 2025 Outward Faster Payment McMillan Alloys Ltd
NA
08 SEP 2025 Direct Debit MR B HAMER
09 SEP 2025 Inward Payment
SD Vehicles

Money out (£)
1,000.00
500.00

Money in (£) Balance (£)
5,000.00 15,000.00
14,000.00
13,500.00
2,000.00 15,500.00`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should parse 4 transactions from the column-separated format
	if len(info.Transactions) != 4 {
		t.Fatalf("transactions: got %d, want 4; parsed: %+v", len(info.Transactions), info.Transactions)
	}

	// Log all transactions for debugging
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	tests := []struct {
		idx     int
		date    string
		typ     string
		amount  float64
		balance float64
	}{
		{0, "05 SEP 2025", "CREDIT", 5000.00, 15000.00},
		{1, "05 SEP 2025", "DEBIT", 1000.00, 14000.00},
		{2, "08 SEP 2025", "DEBIT", 500.00, 13500.00},
		{3, "09 SEP 2025", "CREDIT", 2000.00, 15500.00},
	}

	for _, tt := range tests {
		txn := info.Transactions[tt.idx]
		if txn.Date != tt.date {
			t.Errorf("txn[%d].Date: got %q, want %q", tt.idx, txn.Date, tt.date)
		}
		if txn.Type != tt.typ {
			t.Errorf("txn[%d].Type: got %q, want %q", tt.idx, txn.Type, tt.typ)
		}
		if txn.Amount != tt.amount {
			t.Errorf("txn[%d].Amount: got %.2f, want %.2f", tt.idx, txn.Amount, tt.amount)
		}
		if txn.Balance != tt.balance {
			t.Errorf("txn[%d].Balance: got %.2f, want %.2f", tt.idx, txn.Balance, tt.balance)
		}
	}

	// Verify multi-line descriptions were joined
	if !strings.Contains(info.Transactions[0].Description, "sd vehicles") {
		t.Errorf("txn[0] should include continuation 'sd vehicles': got %q", info.Transactions[0].Description)
	}
	if !strings.Contains(info.Transactions[3].Description, "SD Vehicles") {
		t.Errorf("txn[3] should include continuation 'SD Vehicles': got %q", info.Transactions[3].Description)
	}
}

func TestMetroBankParser_MixedPagesInlineAndColumn(t *testing.T) {
	p := &MetroBankParser{}

	// Tests that a multi-page statement where some pages use inline format
	// and others use column-separated format works correctly.
	pages := []string{
		// Page 1: inline format (amounts on same line as descriptions)
		`Business Bank Account Statement
ACCOUNT NAME: AURORA CAR SALES LTD
From: 01 SEP 2025 To: 30 SEP 2025 Account number 56354379
Sort code 23-05-80

Date Transaction Money out (£) Money in (£) Balance (£)

Balance brought forward 7,225.15

01 SEP 2025 Inward Payment 12,495.00 19,720.15
sd vehicles

02 SEP 2025 Outward Faster Payment SD VEHICLES 1.00 19,719.15
NA`,
		// Page 2: column-separated format (amounts in separate blocks)
		`Date Transaction

05 SEP 2025 Inward Payment
sd vehicles
05 SEP 2025 Outward Faster Payment McMillan Alloys Ltd
NA

Money out (£)
744.00

Money in (£) Balance (£)
15,995.00 16,780.15
16,036.15`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Log all transactions
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	// Page 1: 2 inline transactions + Page 2: 2 column-separated transactions
	if len(info.Transactions) != 4 {
		t.Fatalf("transactions: got %d, want 4; parsed: %+v", len(info.Transactions), info.Transactions)
	}

	// Page 1 txn 0: Inward Payment (credit, inline)
	txn := info.Transactions[0]
	if txn.Type != "CREDIT" {
		t.Errorf("txn[0].Type: got %q, want CREDIT", txn.Type)
	}
	if txn.Amount != 12495.00 {
		t.Errorf("txn[0].Amount: got %.2f, want 12495.00", txn.Amount)
	}

	// Page 1 txn 1: Outward Faster Payment (debit, inline)
	txn = info.Transactions[1]
	if txn.Type != "DEBIT" {
		t.Errorf("txn[1].Type: got %q, want DEBIT", txn.Type)
	}
	if txn.Amount != 1.00 {
		t.Errorf("txn[1].Amount: got %.2f, want 1.00", txn.Amount)
	}

	// Page 2 txn 0: Inward Payment (credit, column-separated)
	txn = info.Transactions[2]
	if txn.Type != "CREDIT" {
		t.Errorf("txn[2].Type: got %q, want CREDIT", txn.Type)
	}
	if txn.Amount != 15995.00 {
		t.Errorf("txn[2].Amount: got %.2f, want 15995.00", txn.Amount)
	}

	// Page 2 txn 1: Outward Faster Payment (debit, column-separated)
	txn = info.Transactions[3]
	if txn.Type != "DEBIT" {
		t.Errorf("txn[3].Type: got %q, want DEBIT", txn.Type)
	}
	if txn.Amount != 744.00 {
		t.Errorf("txn[3].Amount: got %.2f, want 744.00", txn.Amount)
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
