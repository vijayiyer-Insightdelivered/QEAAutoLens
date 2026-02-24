package parser

import (
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// BarclaysParser handles Barclays bank statement PDFs.
//
// Barclays statements come in two main formats:
//
// Format A (standard): Date | Description | Money out | Money in | Balance
//
//	Date format: DD/MM/YYYY or DD Mon YYYY
//	Example: "15/01/2024  CARD PAYMENT TO TESCO STORES 2602  25.99  1,234.56"
//
// Format B (business, arrow-separated): uses → as column separator and short dates "D Mon"
//
//	Example: "5 Dec → Direct Debit to Stripe → 58.80 → 9,397.88"
type BarclaysParser struct{}

func (p *BarclaysParser) BankName() string {
	return "Barclays"
}

// --- Patterns for Format A (standard slash-date Barclays) ---

var barclaysTxnPattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+` +
		`£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})\s*$`,
)

var barclaysTxnSimple = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+£?([\d,]+\.\d{2})\s*$`,
)

// Some Barclays statements use text dates
var barclaysTextDatePattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})\s*$`,
)

// Barclays sometimes uses format: DD Mon  Description  Amount  Balance
var barclaysCompactPattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*)\s+` +
		`(.+?)\s+£?([\d,]+\.\d{2})\s+£?([\d,]+\.\d{2})\s*$`,
)

// --- Pattern for amounts ---

var amountPattern = regexp.MustCompile(`£?([\d,]+\.\d{2})`)

func (p *BarclaysParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankBarclays,
	}

	allText := strings.Join(pages, "\n")

	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractBarclaysName(allText)
	info.StatementPeriod = extractPeriod(allText)

	// Detect if this is an arrow-separated format
	arrowFormat := strings.Contains(allText, "→")

	for _, page := range pages {
		lines := strings.Split(page, "\n")
		var txns []models.Transaction
		if arrowFormat {
			var openBal float64
			txns, openBal = p.parseLinesArrow(lines)
			// Keep the first non-zero opening balance we find
			if info.OpeningBalance == 0 && openBal != 0 {
				info.OpeningBalance = openBal
			}
		} else {
			txns = p.parseLines(lines)
		}
		info.Transactions = append(info.Transactions, txns...)
	}

	return info, nil
}

// parseLinesArrow handles Barclays business statements that use → as column separators
// and short dates (D Mon) without year.
//
// Line examples:
//
//	"4 Dec Start Balance → 9,856.68"
//	"On-Line Banking Bill Payment to → 400.00 → 9,456.68"
//	"5 Dec → Direct Debit to Stripe → 58.80 → 9,397.88"
//	"Direct Credit From Antalis Limited → 10,500.00 19,749.38"
//	"Ref: Antalis Limited" (continuation)
func (p *BarclaysParser) parseLinesArrow(lines []string) ([]models.Transaction, float64) {
	var transactions []models.Transaction
	var openingBalance float64
	inTransactionSection := false
	currentDate := ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Detect header row
		if containsBarclaysHeader(line) {
			inTransactionSection = true
			continue
		}

		// Skip footer/boilerplate
		if isBarclaysFooter(line) {
			continue
		}

		// Skip known non-transaction lines
		if isBarclaysSkipLine(line) {
			continue
		}

		// Handle balance summary lines: extract date and opening balance, then skip
		if isBalanceLine(line) {
			// Extract date from balance line so subsequent dateless transactions
			// inherit the correct date
			if sd := extractShortDate(line); sd != "" {
				currentDate = sd
				inTransactionSection = true
			}
			// Extract opening balance amount (from "Start Balance" or "Balance brought forward")
			if isOpeningBalanceLine(line) && openingBalance == 0 {
				if amounts := amountPattern.FindAllString(line, -1); len(amounts) > 0 {
					if bal, err := parseAmount(amounts[len(amounts)-1]); err == nil {
						openingBalance = bal
					}
				}
			}
			continue
		}

		// Skip summary lines (Total Payments/Receipts, etc.)
		if isSummaryLine(line) {
			continue
		}

		// Skip foreign currency detail lines (continuation info, not transactions)
		if isBarclaysFXDetailLine(line) {
			if len(transactions) > 0 {
				// Append to last transaction description for completeness
				cleanLine := strings.ReplaceAll(line, "→", "")
				cleanLine = strings.TrimSpace(cleanLine)
				last := &transactions[len(transactions)-1]
				last.Description += " " + cleanLine
			}
			continue
		}

		// Check if line starts with a short date (e.g., "4 Dec", "15 Dec")
		shortDate := extractShortDate(line)
		if shortDate != "" {
			currentDate = shortDate
			inTransactionSection = true
		}

		// Also check for full date formats
		if !inTransactionSection && (startsWithDate(line) || shortDate != "") {
			inTransactionSection = true
			if shortDate == "" {
				currentDate = extractDate(line)
			}
		}

		if !inTransactionSection {
			continue
		}

		// Split line by → to get column segments
		parts := strings.Split(line, "→")
		for j := range parts {
			parts[j] = strings.TrimSpace(parts[j])
		}

		// Determine if this is a real transaction line (has amounts in column positions)
		// versus a continuation/detail line that happens to contain numbers
		if !isBarclaysTransactionLine(parts) {
			// No amounts in expected column positions — continuation line
			if len(transactions) > 0 {
				cleanLine := strings.ReplaceAll(line, "→", "")
				cleanLine = strings.TrimSpace(cleanLine)
				if cleanLine != "" && !isBarclaysFooter(cleanLine) {
					last := &transactions[len(transactions)-1]
					last.Description += " " + cleanLine
				}
			}
			continue
		}

		// Extract the transaction from the arrow-separated columns
		txn := parseBarclaysArrowTransaction(parts, shortDate, currentDate)
		if txn != nil {
			transactions = append(transactions, *txn)
		}
	}

	return transactions, openingBalance
}

// isOpeningBalanceLine checks if a balance line represents the opening balance
// (as opposed to "balance carried forward" or "end balance").
func isOpeningBalanceLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "start balance") ||
		strings.Contains(lower, "balance brought forward")
}

// isBarclaysTransactionLine determines if arrow-separated parts represent a real transaction.
// A transaction line has at least one monetary amount in a column position after the description.
// Lines that are just text (Ref:, continuation) or FX detail lines are not transactions.
func isBarclaysTransactionLine(parts []string) bool {
	if len(parts) < 2 {
		return false
	}

	// Check if any part (after the first) is purely a monetary amount
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// A column is a transaction amount if it matches "amount [amount]" format
		fields := strings.Fields(part)
		allAmounts := true
		hasAmount := false
		for _, f := range fields {
			if amountPattern.MatchString(f) {
				hasAmount = true
			} else {
				allAmounts = false
			}
		}
		if allAmounts && hasAmount {
			return true
		}
	}

	return false
}

// parseBarclaysArrowTransaction extracts a transaction from arrow-separated column parts.
func parseBarclaysArrowTransaction(parts []string, shortDate, currentDate string) *models.Transaction {
	if len(parts) < 2 {
		return nil
	}

	// Extract description from the first non-amount part(s)
	desc := extractBarclaysDescription(parts, shortDate)
	if desc == "" {
		return nil
	}

	// Collect all amounts from the column parts (everything after description)
	var amounts []float64
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		for _, f := range strings.Fields(part) {
			if amountPattern.MatchString(f) {
				a, err := parseAmount(f)
				if err == nil && a > 0 {
					amounts = append(amounts, a)
				}
			}
		}
	}

	if len(amounts) == 0 {
		return nil
	}

	txn := &models.Transaction{
		Date:        currentDate,
		Description: desc,
	}

	if len(amounts) >= 2 {
		txn.Amount = amounts[0]
		txn.Balance = amounts[len(amounts)-1]
	} else {
		txn.Amount = amounts[0]
	}

	// Determine debit vs credit
	if isDebitDescription(desc) {
		txn.Type = "DEBIT"
	} else if isCreditDescription(desc) {
		txn.Type = "CREDIT"
	} else {
		// Infer from arrow structure: debits have → between amount and balance,
		// credits have amount and balance in the same column segment (no → between)
		txn.Type = inferTypeFromArrowParts(parts)
	}

	return txn
}

// extractBarclaysDescription gets the description text from arrow-separated parts.
func extractBarclaysDescription(parts []string, shortDate string) string {
	if len(parts) == 0 {
		return ""
	}

	// The first part contains the date (if any) and the start of description
	firstPart := parts[0]

	// Remove the short date from the beginning if present
	if shortDate != "" {
		idx := strings.Index(firstPart, shortDate)
		if idx >= 0 {
			firstPart = strings.TrimSpace(firstPart[idx+len(shortDate):])
		}
	}

	// If the first part is now empty (date was the only thing), look at second part
	if firstPart == "" && len(parts) > 1 {
		// The description might be in the second segment (e.g., "5 Dec → Direct Debit to Stripe → ...")
		desc := parts[1]
		// Remove any amount from this segment
		desc = amountPattern.ReplaceAllString(desc, "")
		return cleanDescription(desc)
	}

	// Check if the first part has amounts — if so, description is before them
	amountLocs := amountPattern.FindAllStringIndex(firstPart, -1)
	if len(amountLocs) > 0 {
		firstPart = firstPart[:amountLocs[0][0]]
	}

	return cleanDescription(firstPart)
}

// cleanDescription trims and normalizes description text.
func cleanDescription(s string) string {
	s = strings.TrimSpace(s)
	// Remove trailing/leading arrows that may remain
	s = strings.Trim(s, "→ ")
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// inferTypeFromArrowParts determines debit/credit based on whether the last
// arrow-separated segment contains one or two amounts.
// Debits: each amount is in its own segment "→ 400.00 → 9,456.68"
// Credits: amount and balance share a segment "→ 10,500.00 19,749.38"
func inferTypeFromArrowParts(parts []string) string {
	if len(parts) < 2 {
		return "DEBIT"
	}

	// Check the last non-empty part
	lastPart := ""
	for i := len(parts) - 1; i >= 1; i-- {
		p := strings.TrimSpace(parts[i])
		if p != "" {
			lastPart = p
			break
		}
	}

	// Count amounts in the last part
	amounts := amountPattern.FindAllString(lastPart, -1)
	if len(amounts) >= 2 {
		// Two amounts in the same column segment = credit (money-in + balance together)
		return "CREDIT"
	}

	return "DEBIT"
}

// isBalanceLine checks if a line refers to a balance summary rather than a real transaction.
func isBalanceLine(line string) bool {
	lower := strings.ToLower(line)
	balanceKeywords := []string{
		"start balance", "balance brought forward", "balance carried forward",
		"end balance",
	}
	for _, kw := range balanceKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isCreditDescription checks if a description indicates an incoming payment.
func isCreditDescription(desc string) bool {
	lower := strings.ToLower(desc)
	creditKeywords := []string{
		"direct credit", "credit from", "bgc ", "bacs ",
		"refund", "interest paid",
		"transfer from", "faster payment",
	}
	for _, kw := range creditKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isBarclaysFXDetailLine identifies foreign currency detail/continuation lines
// that contain amounts but are NOT separate transactions.
// Examples:
//
//	"19.49 On 08 Dec at VISA Exchange Rate 1.33"
//	"The Final GBP Amount Includes A Non-Sterling Transaction Fee of £ 0.40"
//	"USD 69.26 On 01 Jan at VISA Exchange Rate 1.34"
func isBarclaysFXDetailLine(line string) bool {
	lower := strings.ToLower(line)
	fxKeywords := []string{
		"exchange rate",
		"non-sterling transaction fee",
		"final gbp amount",
	}
	for _, kw := range fxKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isBarclaysSkipLine identifies lines that should be skipped during parsing.
func isBarclaysSkipLine(line string) bool {
	lower := strings.ToLower(line)
	skipPhrases := []string{
		"at a glance",
		"your deposit is eligible",
		"compensation scheme",
		"your business current account",
		"issued on",
		"swiftbic",
		"iban gb",
		"anything wrong",
	}
	for _, phrase := range skipPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// --- Format A (standard) parsing — existing logic ---

func (p *BarclaysParser) parseLines(lines []string) []models.Transaction {
	var transactions []models.Transaction
	inTransactionSection := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if containsBarclaysHeader(line) {
			inTransactionSection = true
			continue
		}

		if !inTransactionSection && !startsWithDate(line) {
			continue
		}

		if startsWithDate(line) {
			inTransactionSection = true
		}

		// Try full pattern with slash dates (DD/MM/YYYY)
		if txn, ok := p.tryFullPattern(barclaysTxnPattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try text date pattern (DD Mon YYYY)
		if txn, ok := p.tryFullPattern(barclaysTextDatePattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try compact pattern (DD Mon  Description  Amount  Balance)
		if m := barclaysCompactPattern.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			amt, _ := parseAmount(m[3])
			txn.Amount = amt
			txn.Balance, _ = parseAmount(m[4])

			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Simple pattern: just date + description + one amount
		if m := barclaysTxnSimple.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			amt, _ := parseAmount(m[3])
			txn.Amount = amt
			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Multi-line description continuation
		if len(transactions) > 0 && !startsWithDate(line) && line != "" && inTransactionSection {
			if !isSummaryLine(line) && !isBarclaysFooter(line) {
				last := &transactions[len(transactions)-1]
				last.Description += " " + line
			}
		}
	}

	return transactions
}

func (p *BarclaysParser) tryFullPattern(pat *regexp.Regexp, line string) (models.Transaction, bool) {
	m := pat.FindStringSubmatch(line)
	if m == nil {
		return models.Transaction{}, false
	}

	txn := models.Transaction{
		Date:        m[1],
		Description: strings.TrimSpace(m[2]),
	}

	moneyOut := strings.TrimSpace(m[3])
	moneyIn := strings.TrimSpace(m[4])
	balance := strings.TrimSpace(m[5])

	if moneyOut != "" {
		txn.Amount, _ = parseAmount(moneyOut)
		txn.Type = "DEBIT"
	} else if moneyIn != "" {
		txn.Amount, _ = parseAmount(moneyIn)
		txn.Type = "CREDIT"
	}

	if balance != "" {
		txn.Balance, _ = parseAmount(balance)
	}

	return txn, true
}

func containsBarclaysHeader(line string) bool {
	lower := strings.ToLower(line)
	return (strings.Contains(lower, "date") &&
		(strings.Contains(lower, "money out") || strings.Contains(lower, "money in") ||
			strings.Contains(lower, "description") || strings.Contains(lower, "details"))) ||
		containsTransactionHeader(line)
}

func isBarclaysFooter(line string) bool {
	lower := strings.ToLower(line)
	footerKeywords := []string{
		"barclays bank", "registered in", "authorised by",
		"financial conduct", "please check", "if you find",
		"prudential regulation",
	}
	for _, kw := range footerKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractBarclaysName(text string) string {
	// Barclays often has name at the top of the statement
	name := extractNameNearLabel(text, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms ", "Miss "})
	if name != "" {
		return name
	}

	// Try to find name after sort code/account number line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Sort code") || strings.Contains(line, "Account number") ||
			strings.Contains(line, "Sort Code") || strings.Contains(line, "Account No") {
			if i+1 < len(lines) {
				candidate := strings.TrimSpace(lines[i+1])
				if candidate != "" && !strings.ContainsAny(candidate, "0123456789") {
					return candidate
				}
			}
		}
	}

	return ""
}
