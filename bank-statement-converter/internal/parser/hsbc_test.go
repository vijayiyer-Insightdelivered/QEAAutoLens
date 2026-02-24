package parser

import (
	"strings"
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

func TestHSBCParser_TabSeparated(t *testing.T) {
	p := &HSBCParser{}

	// Simulate pdf.js output with tab-separated columns
	pages := []string{
		"HSBC UK Bank plc\n" +
			"Account name: Jane Doe\n" +
			"Sort code: 40-12-34\tAccount number: 87654321\n" +
			"Date\tPayment type and details\tPaid out\tPaid in\tBalance\n" +
			"01 Jan 24\tBALANCE BROUGHT FORWARD\t\t\t5,000.00\n" +
			"02 Jan 24\tCARD PAYMENT TO TESCO STORES\t25.99\t\t4,974.01\n" +
			"03 Jan 24\tDIRECT DEBIT SKY UK LIMITED\t45.00\t\t4,929.01\n" +
			"05 Jan 24\tSALARY FROM EMPLOYER LTD\t\t2,500.00\t7,429.01\n" +
			"10 Jan 24\tATM WITHDRAWAL LONDON\t100.00\t\t7,329.01\n" +
			"31 Jan 24\tBALANCE CARRIED FORWARD\t\t\t7,329.01",
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Transactions) < 4 {
		t.Fatalf("expected at least 4 transactions, got %d", len(info.Transactions))
	}

	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %s | %s | %.2f | %.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	// Verify balance inference: TESCO should be DEBIT (balance went down)
	for _, txn := range info.Transactions {
		if txn.Description == "CARD PAYMENT TO TESCO STORES" {
			if txn.Type != "DEBIT" {
				t.Errorf("TESCO: expected DEBIT, got %s", txn.Type)
			}
			if txn.Amount != 25.99 {
				t.Errorf("TESCO: expected amount 25.99, got %.2f", txn.Amount)
			}
		}
		if txn.Description == "SALARY FROM EMPLOYER LTD" {
			if txn.Type != "CREDIT" {
				t.Errorf("SALARY: expected CREDIT, got %s", txn.Type)
			}
		}
	}
}

// Test tab format where descriptions may be split across multiple tab cells
func TestHSBCParser_TabSplitDescription(t *testing.T) {
	p := &HSBCParser{}

	// pdf.js might split description text into separate cells
	pages := []string{
		"Date\tPayment type and details\tPaid out\tPaid in\tBalance\n" +
			"15 Jan 24\tCARD PAYMENT\tTO TESCO STORES\t25.99\t1,234.56\n" +
			"16 Jan 24\tDIRECT DEBIT\tSKY UK\tLIMITED\t45.00\t1,189.56",
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %q | %s | %.2f | %.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	if len(info.Transactions) < 2 {
		t.Fatalf("expected 2 transactions, got %d", len(info.Transactions))
	}
}

// Test with actual HSBC PDF output (spread chars, stray "A", dot placeholders)
func TestHSBCParser_RealHSBCFormat(t *testing.T) {
	p := &HSBCParser{}

	// Exact format from actual HSBC PDF via pdf.js extraction
	pages := []string{
		"HSBC UK Bank plc\n" +
			"Account Nam e\tS ortcode\tAccount Num ber\n" +
			"THINKWISE VENTURES LIMITED\t40-21-27\t11623176\n" +
			"Date\tPay m e nt t y pe and de t ails\tPaid out\tPaid in\tBalance\n" +
			"A 30 Dec 25\tBALANCE BROUGHT FORWARD\t.\t5,107.87\n" +
			"30 Jan 26\tCR GROSS INTEREST TO 29JAN2026\t6.07\t5,113.94\n" +
			"30 Jan 26\tBALANCE CARRIED FORWARD\t5,113.94",
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %q | %s | %.2f | %.2f",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance)
	}

	if len(info.Transactions) < 3 {
		t.Fatalf("expected at least 3 transactions, got %d", len(info.Transactions))
	}

	// Verify descriptions are clean (no "25" or "." artifacts)
	for _, txn := range info.Transactions {
		if strings.Contains(txn.Description, " .") || strings.HasPrefix(txn.Description, "25 ") {
			t.Errorf("description has artifacts: %q", txn.Description)
		}
	}

	// Verify the interest transaction is present
	found := false
	for _, txn := range info.Transactions {
		if strings.Contains(txn.Description, "INTEREST") {
			found = true
			if txn.Amount != 6.07 {
				t.Errorf("interest amount: got %.2f, want 6.07", txn.Amount)
			}
			if txn.Type != "CREDIT" {
				t.Errorf("interest type: got %s, want CREDIT", txn.Type)
			}
		}
	}
	if !found {
		t.Error("interest transaction not found")
	}
}

// Test the EXACT real-world scenario: interest line split across two PDF lines
func TestHSBCParser_SplitLineJoin(t *testing.T) {
	p := &HSBCParser{}

	// Exact lines from the debug output:
	// Line 30: "30 Jan 26\tCR GROSS INTEREST" (date + partial desc, NO amounts)
	// Line 31: "TO 29JAN2026\t6.07\t5,113.94" (rest of desc + amounts, NO date)
	pages := []string{
		"Date\tPayment type and details\tPaid out\tPaid in\tBalance\n" +
			"30 Dec 25\tBALANCE BROUGHT FORWARD\t.\t5,107.87\n" +
			"30 Jan 26\tCR GROSS INTEREST\n" +
			"TO 29JAN2026\t6.07\t5,113.94\n" +
			"30 Jan 26\tBALANCE CARRIED FORWARD\t5,113.94",
	}

	info, err := p.Parse(pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("parsed %d transactions", len(info.Transactions))
	for i, txn := range info.Transactions {
		t.Logf("  [%d] %s | %q | %s | %.2f | %.2f | method=%s",
			i, txn.Date, txn.Description, txn.Type, txn.Amount, txn.Balance, txn.ParseMethod)
	}

	if len(info.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(info.Transactions))
	}

	// Verify the joined interest transaction
	interest := info.Transactions[1]
	if !strings.Contains(interest.Description, "INTEREST") {
		t.Errorf("expected interest transaction at index 1, got %q", interest.Description)
	}
	if interest.Amount != 6.07 {
		t.Errorf("interest amount: got %.2f, want 6.07", interest.Amount)
	}
	if interest.Balance != 5113.94 {
		t.Errorf("interest balance: got %.2f, want 5113.94", interest.Balance)
	}
	if interest.ParseMethod != "tab-separated-joined" {
		t.Errorf("expected parse method 'tab-separated-joined', got %q", interest.ParseMethod)
	}

	// Verify BALANCE BROUGHT FORWARD was NOT polluted with continuation text
	bf := info.Transactions[0]
	if strings.Contains(bf.Description, "29JAN") || strings.Contains(bf.Description, "6.07") {
		t.Errorf("BALANCE BROUGHT FORWARD has leaked continuation: %q", bf.Description)
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
