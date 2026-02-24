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

func TestBarclaysParser_ArrowFormat(t *testing.T) {
	p := &BarclaysParser{}

	// Realistic Barclays business statement with → separators and short dates
	pages := []string{
		`INSIGHT DELIVERED LIMITED Sort Code 20-71-03 Account No 90950467
SWIFTBIC BUKBGB22 IBAN GB29 BUKB 2071 0390 9504 67
Issued on 05 January 2026
MR KULBIR MINHAS
INSIGHT DELIVERED LIMITED
1 PAPERMILL AVENUE
HOOK RG27 9QU
Your Business Current Account → At a glance
04 Dec 2025 - 02 Jan
Date Description → Money out £ Money in £ → Balance £
2026
4 Dec Start Balance → 9,856.68
On-Line Banking Bill Payment to → 400.00 → 9,456.68
Mads Rose Trading
Ref: Inv 1
5 Dec → Direct Debit to Stripe → 58.80 → 9,397.88
Ref: 7Trknzzm-SL
Commission Charges For The → 8.50 → 9,389.38
Period 13 Oct /12 Nov
8 Dec → On-Line Banking Bill Payment to → 140.00 → 9,249.38
Sasha Mitchell
Ref: Invs 1631
Direct Credit From Antalis Limited → 10,500.00 19,749.38
Ref: Antalis Limited`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.SortCode != "20-71-03" {
		t.Errorf("sort code: got %q, want %q", info.SortCode, "20-71-03")
	}

	if info.AccountNumber != "90950467" {
		t.Errorf("account number: got %q, want %q", info.AccountNumber, "90950467")
	}

	// Verify opening balance is captured
	if info.OpeningBalance != 9856.68 {
		t.Errorf("opening balance: got %.2f, want 9856.68", info.OpeningBalance)
	}

	t.Logf("opening balance: %.2f", info.OpeningBalance)
	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	if len(info.Transactions) < 5 {
		t.Fatalf("expected at least 5 transactions (including Start Balance), got %d", len(info.Transactions))
	}

	// Verify Start Balance is emitted as a BALANCE transaction
	found := false
	for _, txn := range info.Transactions {
		if txn.Type == "BALANCE" && txn.Balance == 9856.68 {
			found = true
			if txn.Date != "4 Dec" {
				t.Errorf("Start Balance date: got %q, want %q", txn.Date, "4 Dec")
			}
			break
		}
	}
	if !found {
		t.Error("expected to find Start Balance transaction with balance 9,856.68")
	}

	// Verify the bill payment has the correct date from the "Start Balance" line
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 400.00 && txn.Type == "DEBIT" {
			found = true
			if txn.Date != "4 Dec" {
				t.Errorf("Mads Rose Trading txn date: got %q, want %q", txn.Date, "4 Dec")
			}
			if txn.Balance != 9456.68 {
				t.Errorf("Mads Rose Trading txn balance: got %.2f, want 9456.68", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find bill payment of 400.00 (debit)")
	}

	// Transaction 2: Direct Debit to Stripe (debit)
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 58.80 && txn.Type == "DEBIT" {
			found = true
			if txn.Balance != 9397.88 {
				t.Errorf("Stripe txn balance: got %.2f, want 9397.88", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find Stripe direct debit of 58.80")
	}

	// Transaction 3: Direct Credit from Antalis (credit)
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 10500.00 && txn.Type == "CREDIT" {
			found = true
			if txn.Balance != 19749.38 {
				t.Errorf("Antalis credit balance: got %.2f, want 19749.38", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find Antalis credit of 10,500.00")
	}
}

func TestBarclaysParser_ArrowFormat_Page2(t *testing.T) {
	p := &BarclaysParser{}

	// Page 2 with "Balance brought forward from previous page"
	pages := []string{
		`Insight Delivered Limited • Sort Code 20-71-03 • Account No 90950467
Date Description → Money out £ → Money in £ → Balance £
BalanceBalance brought forward from previous pagebrought forward from previous page → 13,234.35
12 Dec → On-Line Banking Bill Payment to → 656.25 → 12,578.10
Hidden Gem -
Your Ref: 379
On-Line Banking Bill Payment to → 800.00 → 11,778.10
Business Marketing
Ref: Inv-0153
On-Line Banking Bill Payment to → 910.00 → 10,868.10
Gillian Perkins
Ref: ID11 25
On-Line Banking Bill Payment to → 1,555.20 → 9,312.90
Zoho Corporation L
Ref: 80030737171
15 Dec → Card Payment to Lebara Mobile → 6.90 → 9,306.00
Limi On 14 Dec
Card Payment to → 86.52 → 9,219.48
Microsoft#G1296809 On 13 Dec
Card Payment to → 99.96 → 9,119.52
Microsoft-G1296880 On 13 Dec
On-Line Banking Bill Payment to → 7,920.00 → 1,199.52
Thinkviz
Ref: Inv 1150
Direct Credit From Antalis Limited → 10,500.00 11,699.52
Ref: Antalis Limited
16 Dec → Card Payment to Dialpad Inc USA → 29.16 → 11,670.36
On 15 Dec
17 Dec → On-Line Banking Bill Payment to → 57.20 → 11,613.16
Ian Malcolm Kerr
Ref: Rail Ticket
On-Line Banking Bill Payment to → 393.56 → 11,219.60
Vijay Muralidharan
Ref: Team Expenses
19 Dec → On-Line Banking Bill Payment to → 772.17 → 10,447.43
HMRC PAYE/Nic Cumb
Ref: 120PK012490842607
Card Payment to Linkedin SN → 66.66 → 10,380.77
P99858 Ireland On 18 Dec
29 Dec → Standing Order to Venugopal → 812.50 → 9,568.27
Lakshman
Ref:- Insight Salary
On-Line Banking Bill Payment to → 4,746.97 → 4,821.30
Alexander James CA
Ref: Dec Salary
30 Dec → On-Line Banking Bill Payment to → 19.20 → 4,802.10
Idos Virtual Servi
Ref: 002443
On-Line Banking Bill Payment to → 583.51 → 4,218.59
Centrilogic Ltd
Ref: UK56039
31 Dec → On-Line Banking Bill Payment to → 438.36 → 3,780.23
Peter Robertson
Ref: Inv 2484`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify opening balance from "Balance brought forward" line
	if info.OpeningBalance != 13234.35 {
		t.Errorf("opening balance: got %.2f, want 13234.35", info.OpeningBalance)
	}

	t.Logf("opening balance: %.2f", info.OpeningBalance)
	t.Logf("parsed %d transactions from page 2", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	// Should parse many transactions from this page
	if len(info.Transactions) < 15 {
		t.Errorf("expected at least 15 transactions, got %d", len(info.Transactions))
	}

	// Verify Antalis credit on page 2
	found := false
	for _, txn := range info.Transactions {
		if txn.Amount == 10500.00 && txn.Type == "CREDIT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Antalis credit of 10,500.00 on page 2")
	}

	// Verify HMRC payment (debit)
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 772.17 && txn.Type == "DEBIT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find HMRC payment of 772.17")
	}
}

func TestBarclaysParser_ArrowFormat_Page3(t *testing.T) {
	p := &BarclaysParser{}

	// Page 3 with foreign currency transaction and final balance
	pages := []string{
		`Insight Delivered Limited • Sort Code 20-71-03 • Account No 90950467
Date Description → Money out £ → Money in £ → Balance £
BalanceBalance brought forward from previous pagebrought forward from previous page → 3,780.23
2 Jan → Card Payment to → 53.11 → 3,727.12
Digitalocean.Com USD 69.26 On 01 Jan at VISA Exchange Rate 1.34
The Final GBP Amount Includes A Non-Sterling Transaction Fee of £ 1.42
2 Jan Balance carried forward → 3,727.12
Total Payments/Receipts → 27,129.56 21,000.00`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("parsed %d transactions from page 3", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	// Should find the DigitalOcean transaction
	found := false
	for _, txn := range info.Transactions {
		if txn.Amount == 53.11 && txn.Type == "DEBIT" {
			found = true
			if txn.Balance != 3727.12 {
				t.Errorf("DigitalOcean balance: got %.2f, want 3727.12", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find DigitalOcean payment of 53.11")
	}
}

func TestBarclaysParser_SharedDateFormat(t *testing.T) {
	p := &BarclaysParser{}

	// Barclays business statement extracted without → arrows.
	// Dates are DD Mon (no year) and shared across multiple transactions.
	pages := []string{
		`INSIGHT DELIVERED LIMITED Sort Code 20-71-03 Account No 90950467
Issued on 05 January 2026
Date Description Money out Money in Balance
4 Dec Start Balance 9,856.68
On-Line Banking Bill Payment to Mads Rose Trading 400.00 9,456.68
Ref: Inv 1
5 Dec Direct Debit to Stripe 58.80 9,397.88
Ref: 7Trknzzm-SL
Direct Credit From Antalis Limited 800.00 10,197.88
8 Dec Direct Debit to Sage UK Ltd 84.00 10,113.88
On-Line Banking Bill Payment to UK Insurance 400.00 9,713.88
Direct Credit From Antalis Limited 10,500.00 20,213.88
9 Dec Card Payment Amazon 14.99 20,198.89`,
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("parsed %d transactions (shared-date format)", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] date=%q desc=%q type=%s amount=%.2f balance=%.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	// Should parse at least 8 transactions (including Start Balance)
	if len(info.Transactions) < 8 {
		t.Fatalf("expected at least 8 transactions, got %d", len(info.Transactions))
	}

	// Verify OpeningBalance is captured
	if info.OpeningBalance != 9856.68 {
		t.Errorf("opening balance: got %.2f, want 9856.68", info.OpeningBalance)
	}

	// Verify Start Balance transaction
	found := false
	for _, txn := range info.Transactions {
		if txn.Type == "BALANCE" && txn.Balance == 9856.68 {
			found = true
			if txn.Date != "4 Dec" {
				t.Errorf("Start Balance date: got %q, want %q", txn.Date, "4 Dec")
			}
			break
		}
	}
	if !found {
		t.Error("expected to find Start Balance with balance 9,856.68")
	}

	// Verify bill payment on same date as Start Balance (shared date "4 Dec")
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 400.00 && txn.Date == "4 Dec" && txn.Type == "DEBIT" {
			found = true
			if txn.Balance != 9456.68 {
				t.Errorf("Mads Rose balance: got %.2f, want 9456.68", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected bill payment of 400.00 on 4 Dec (shared date)")
	}

	// Verify transaction under "5 Dec" with no date prefix (inherited date)
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 800.00 && txn.Type == "CREDIT" {
			found = true
			if txn.Date != "5 Dec" {
				t.Errorf("Antalis 800.00 credit date: got %q, want %q", txn.Date, "5 Dec")
			}
			break
		}
	}
	if !found {
		t.Error("expected Direct Credit of 800.00 under shared date 5 Dec")
	}

	// Verify "8 Dec" has 3 transactions
	count8Dec := 0
	for _, txn := range info.Transactions {
		if txn.Date == "8 Dec" && txn.Type != "BALANCE" {
			count8Dec++
		}
	}
	if count8Dec != 3 {
		t.Errorf("expected 3 transactions on 8 Dec, got %d", count8Dec)
	}

	// Verify Antalis credit on 8 Dec
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 10500.00 && txn.Type == "CREDIT" && txn.Date == "8 Dec" {
			found = true
			if txn.Balance != 20213.88 {
				t.Errorf("Antalis 10,500 balance: got %.2f, want 20213.88", txn.Balance)
			}
			break
		}
	}
	if !found {
		t.Error("expected Antalis credit of 10,500.00 on 8 Dec")
	}

	// Verify 9 Dec transaction
	found = false
	for _, txn := range info.Transactions {
		if txn.Amount == 14.99 && txn.Date == "9 Dec" && txn.Type == "DEBIT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Card Payment Amazon of 14.99 on 9 Dec")
	}
}
