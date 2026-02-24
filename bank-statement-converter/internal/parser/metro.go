package parser

import (
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// MetroBankParser handles Metro Bank statement PDFs.
//
// Metro Bank statements typically have this layout:
//
//	Date | Transaction type | Description | Paid out | Paid in | Balance
//
// Date format: DD/MM/YYYY (digital PDF) or DD MMM YYYY (OCR scanned)
// Example line: "15/01/2024 CARD PAYMENT TESCO STORES 25.99 1,234.56"
// OCR example:  "01 SEP 2025 Inward Payment sd vehicles 12,495.00 19,720.15"
type MetroBankParser struct{}

func (p *MetroBankParser) BankName() string {
	return "Metro Bank"
}

// Metro Bank transaction line patterns — DD/MM/YYYY format (digital PDF):
var metroTxnPattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)` +
		`\s+([\d,]+\.\d{2})?\s*([\d,]+\.\d{2})?\s+([\d,]+\.\d{2})\s*$`,
)

var metroTxnSimple = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`,
)

// OCR format patterns — DD MMM YYYY format (from Tesseract OCR):
// Full: date + description + up to 3 amounts (paid out, paid in, balance)
var metroOCRFull = regexp.MustCompile(
	`(?i)^(\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\s+\d{4})\s+(.+?)` +
		`\s+([\d,]+\.\d{2})\s+([\d,]+\.\d{2})\s*$`,
)

// OCR: date + description + single amount
var metroOCRSimple = regexp.MustCompile(
	`(?i)^(\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\s+\d{4})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`,
)

// OCR: date + description only (no amounts — amounts on separate section)
var metroOCRDateOnly = regexp.MustCompile(
	`(?i)^(\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\s+\d{4})\s+(.+?)\s*$`,
)

// Matches a standalone amount line (used in separated OCR columns)
var amountLinePattern = regexp.MustCompile(`^[\d,]+\.\d{2}(?:\s+[\d,]+\.\d{2})*\s*$`)

func (p *MetroBankParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankMetro,
	}

	allText := strings.Join(pages, "\n")

	// Extract account metadata
	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractMetroAccountName(allText)
	info.StatementPeriod = extractPeriod(allText)

	for _, page := range pages {
		// Sanitize OCR errors before parsing
		page = sanitizeOCRAmounts(page)
		lines := strings.Split(page, "\n")
		txns := p.parseLines(lines)
		info.Transactions = append(info.Transactions, txns...)
	}

	return info, nil
}

func (p *MetroBankParser) parseLines(lines []string) []models.Transaction {
	var transactions []models.Transaction
	inTransactionSection := false

	// Two-pass approach for OCR pages where amounts are in a separate section:
	// Pass 1: collect transactions (with or without amounts)
	// Pass 2: if we find an amount section, match amounts to transactions without them

	var noAmountTxns []*models.Transaction // transactions that need amounts
	amountSectionStart := -1

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Detect start of transaction table
		if containsTransactionHeader(line) {
			inTransactionSection = true
			continue
		}

		// Detect separated amount section header (OCR artifact)
		if isMoneyHeader(line) {
			amountSectionStart = i + 1
			continue
		}

		// Skip non-transaction lines before the table
		if !inTransactionSection && !startsWithDate(line) {
			continue
		}

		if startsWithDate(line) {
			inTransactionSection = true
		}

		// Skip known non-transaction content
		if isSummaryLine(line) || isFooterLine(line) {
			continue
		}

		// Try DD/MM/YYYY patterns first (digital PDF)
		if m := metroTxnPattern.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			paidOut := strings.TrimSpace(m[3])
			paidIn := strings.TrimSpace(m[4])
			balance := strings.TrimSpace(m[5])

			if paidOut != "" {
				txn.Amount, _ = parseAmount(paidOut)
				txn.Type = "DEBIT"
			} else if paidIn != "" {
				txn.Amount, _ = parseAmount(paidIn)
				txn.Type = "CREDIT"
			}
			if balance != "" {
				txn.Balance, _ = parseAmount(balance)
			}
			transactions = append(transactions, txn)
			continue
		}

		if m := metroTxnSimple.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			txn.Amount, _ = parseAmount(m[3])
			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Try OCR DD MMM YYYY patterns
		// Full pattern: date + description + 2 amounts (amount + balance)
		if m := metroOCRFull.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: cleanOCRDescription(m[2]),
			}
			amt1, _ := parseAmount(m[3])
			amt2, _ := parseAmount(m[4])

			// Determine debit/credit from description
			if isDebitDescription(txn.Description) {
				txn.Amount = amt1
				txn.Type = "DEBIT"
				txn.Balance = amt2
			} else if isInwardDescription(txn.Description) {
				txn.Amount = amt1
				txn.Type = "CREDIT"
				txn.Balance = amt2
			} else {
				// Heuristic: if amt2 > amt1, amt1 is the transaction amount
				txn.Amount = amt1
				txn.Balance = amt2
				if isDebitDescription(txn.Description) {
					txn.Type = "DEBIT"
				} else {
					txn.Type = "CREDIT"
				}
			}
			transactions = append(transactions, txn)
			continue
		}

		// Simple OCR pattern: date + description + 1 amount
		if m := metroOCRSimple.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: cleanOCRDescription(m[2]),
			}
			txn.Amount, _ = parseAmount(m[3])
			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else if isInwardDescription(txn.Description) {
				txn.Type = "CREDIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// OCR date-only pattern (amounts in separate section)
		if m := metroOCRDateOnly.FindStringSubmatch(line); m != nil {
			desc := cleanOCRDescription(m[2])
			// Skip if description looks like non-transaction content
			if isMetroNonTransaction(desc) {
				continue
			}
			txn := models.Transaction{
				Date:        m[1],
				Description: desc,
			}
			if isDebitDescription(desc) {
				txn.Type = "DEBIT"
			} else if isInwardDescription(desc) {
				txn.Type = "CREDIT"
			} else {
				txn.Type = "DEBIT" // default for Metro outward payments
			}
			transactions = append(transactions, txn)
			noAmountTxns = append(noAmountTxns, &transactions[len(transactions)-1])
			continue
		}

		// Handle multi-line descriptions
		if len(transactions) > 0 && !startsWithDate(line) && line != "" && inTransactionSection {
			if !isSummaryLine(line) && !isFooterLine(line) && !amountLinePattern.MatchString(line) && !isMoneyHeader(line) {
				last := &transactions[len(transactions)-1]
				last.Description += " " + line
			}
		}
	}

	// Pass 2: Match amounts from separated section to transactions without amounts
	if amountSectionStart > 0 && len(noAmountTxns) > 0 {
		p.matchSeparatedAmounts(lines[amountSectionStart:], noAmountTxns)
	}

	return transactions
}

// matchSeparatedAmounts pairs amount lines with transactions that have no amounts.
// In OCR output, wide tables get split: descriptions on top, amounts at bottom.
func (p *MetroBankParser) matchSeparatedAmounts(amountLines []string, txns []*models.Transaction) {
	var amounts []amountEntry
	for _, line := range amountLines {
		line = strings.TrimSpace(line)
		if line == "" || isSummaryLine(line) || isFooterLine(line) {
			continue
		}
		// Parse amounts from the line
		amountRe := regexp.MustCompile(`[\d,]+\.\d{2}`)
		matches := amountRe.FindAllString(line, -1)
		if len(matches) == 0 {
			continue
		}

		entry := amountEntry{}
		if len(matches) == 1 {
			entry.amount, _ = parseAmount(matches[0])
		} else if len(matches) == 2 {
			entry.amount, _ = parseAmount(matches[0])
			entry.balance, _ = parseAmount(matches[1])
		} else if len(matches) >= 3 {
			entry.paidOut, _ = parseAmount(matches[0])
			entry.paidIn, _ = parseAmount(matches[1])
			entry.balance, _ = parseAmount(matches[2])
		}
		amounts = append(amounts, entry)
	}

	// Match amounts to transactions by position
	// The amount section has paired entries: money out/in on one line, balance on next
	// We need to figure out the pattern. Try direct 1:1 matching first.
	if len(amounts) >= len(txns) {
		for i, txn := range txns {
			if i >= len(amounts) {
				break
			}
			a := amounts[i]
			if a.paidOut > 0 {
				txn.Amount = a.paidOut
				txn.Type = "DEBIT"
			} else if a.paidIn > 0 {
				txn.Amount = a.paidIn
				txn.Type = "CREDIT"
			} else if a.amount > 0 {
				txn.Amount = a.amount
				// Keep existing type from description analysis
			}
			if a.balance > 0 {
				txn.Balance = a.balance
			}
		}
	}
}

type amountEntry struct {
	amount  float64
	paidOut float64
	paidIn  float64
	balance float64
}

// isMoneyHeader detects the "Money out (£) Money in (£) Balance (£)" header
// that separates descriptions from amounts in OCR output.
func isMoneyHeader(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "money out") && strings.Contains(lower, "money in")
}

// isInwardDescription checks if description indicates an incoming payment.
func isInwardDescription(desc string) bool {
	lower := strings.ToLower(desc)
	return strings.Contains(lower, "inward") || strings.Contains(lower, "credit") ||
		strings.Contains(lower, "received") || strings.Contains(lower, "refund") ||
		strings.Contains(lower, "interest paid")
}

// cleanOCRDescription cleans up OCR artifacts from descriptions.
func cleanOCRDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	// Remove leading/trailing quote marks from OCR
	desc = strings.TrimLeft(desc, "'\"'")
	desc = strings.TrimRight(desc, "'\"'")
	// Collapse multiple spaces
	desc = regexp.MustCompile(`\s{2,}`).ReplaceAllString(desc, " ")
	return desc
}

// isMetroNonTransaction checks if a description is actually non-transaction content
// that happens to start with a date-like pattern.
func isMetroNonTransaction(desc string) bool {
	lower := strings.ToLower(desc)
	nonTxnPhrases := []string{
		"balance brought forward", "balance carried forward",
		"your account summary", "account name",
		"opening balance", "closing balance", "end balance",
		"statement number", "overdraft limit",
		"total money", "important information",
		"metro bank pl", "registered in england",
	}
	for _, phrase := range nonTxnPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func containsTransactionHeader(line string) bool {
	lower := strings.ToLower(line)
	// "paid" is included in the description check because HSBC PDFs use spread
	// characters in headers (e.g. "Pay m e nt t y pe and de t ails") which makes
	// "details" undetectable, but "Paid out" column header remains intact
	return strings.Contains(lower, "date") &&
		(strings.Contains(lower, "description") || strings.Contains(lower, "transaction") ||
			strings.Contains(lower, "details") || strings.Contains(lower, "paid")) &&
		(strings.Contains(lower, "amount") || strings.Contains(lower, "paid") ||
			strings.Contains(lower, "balance") || strings.Contains(lower, "money"))
}

func isDebitDescription(desc string) bool {
	lower := strings.ToLower(desc)
	debitKeywords := []string{
		"card payment", "card purchase", "direct debit", "debit", "payment",
		"withdrawal", "transfer out", "standing order", "dd ", "pos ", "atm ",
		"purchase", "fee", "charge", "outward", "internet banking chg",
		"account maintenance", "transaction charge",
	}
	for _, kw := range debitKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isSummaryLine(line string) bool {
	lower := strings.ToLower(line)
	summaryKeywords := []string{
		"opening balance", "closing balance", "total paid in",
		"total paid out", "total payments", "total receipts",
		"statement period", "page ", "continued",
		"total fees and charges", "monthly maintenance",
		"total money", "end balance",
	}
	for _, kw := range summaryKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isFooterLine detects bank footer/legal text that appears on every page.
func isFooterLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "metro bank pl") ||
		strings.Contains(lower, "registered in england") ||
		strings.Contains(lower, "financial conduct") ||
		strings.Contains(lower, "prudential regulation") ||
		strings.Contains(lower, "compensation scheme") ||
		strings.Contains(lower, "we love to hear") ||
		strings.Contains(lower, "listening to you") ||
		strings.Contains(lower, "financial ombudsman")
}

func extractMetroAccountName(text string) string {
	// Try "ACCOUNT NAME:" label first (OCR output)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToUpper(line), "ACCOUNT NAME") {
			rest := line
			if idx := strings.Index(strings.ToUpper(rest), "ACCOUNT NAME"); idx >= 0 {
				rest = rest[idx+len("ACCOUNT NAME"):]
			}
			rest = strings.TrimLeft(rest, ": ")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				return rest
			}
		}
	}
	// Fall back to generic label search
	return extractNameNearLabel(text, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms "})
}

func extractNameNearLabel(text string, labels []string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		for _, label := range labels {
			if idx := strings.Index(line, label); idx >= 0 {
				rest := strings.TrimSpace(line[idx+len(label):])
				if colonIdx := strings.Index(rest, ":"); colonIdx == 0 {
					rest = strings.TrimSpace(rest[1:])
				}
				if rest != "" {
					parts := strings.Split(rest, "  ")
					return strings.TrimSpace(parts[0])
				}
			}
		}
	}
	return ""
}

func extractPeriod(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "statement period") || strings.Contains(lower, "period") {
			dates := datePatternSlash.FindAllString(line, 2)
			if len(dates) == 2 {
				return dates[0] + " to " + dates[1]
			}
			textDates := datePatternText.FindAllString(line, 2)
			if len(textDates) == 2 {
				return textDates[0] + " to " + textDates[1]
			}
		}
		// Also check "From: DD MMM YYYY To: DD MMM YYYY" (OCR format)
		if strings.Contains(lower, "from:") && strings.Contains(lower, "to:") {
			textDates := datePatternText.FindAllString(line, 2)
			if len(textDates) == 2 {
				return textDates[0] + " to " + textDates[1]
			}
		}
	}
	return ""
}
