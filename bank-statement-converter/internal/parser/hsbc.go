package parser

import (
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// HSBCParser handles HSBC bank statement PDFs.
//
// HSBC statements typically have this layout:
//
//	Date | Payment type and details | Paid out | Paid in | Balance
//
// Date format: DD Mon YY (e.g., 15 Jan 24) or DD Mon YYYY
type HSBCParser struct{}

func (p *HSBCParser) BankName() string {
	return "HSBC"
}

// amountCellPattern matches a cell containing a single monetary amount.
var amountCellPattern = regexp.MustCompile(`^[£\x{00A3}]?\s*([\d,]+\.\d{2})\s*$`)

// HSBC transaction line patterns (for non-tab-separated text)
var hsbcTxnPattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s{2,}` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s+[£\x{00A3}]?([\d,]+\.\d{2})?\s+[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

var hsbcTxnFlexible = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

var hsbcTxnSimple = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

var hsbcDashDatePattern = regexp.MustCompile(
	`^(\d{1,2}-(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*-\d{2,4})\s+` +
		`(.+?)\s+[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

var hsbcSlashDatePattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

func (p *HSBCParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankHSBC,
	}

	allText := strings.Join(pages, "\n")

	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractNameNearLabel(allText, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms ", "Name"})
	info.StatementPeriod = extractPeriod(allText)

	for _, page := range pages {
		lines := strings.Split(page, "\n")
		txns, debugLines := p.parseLines(lines)
		info.Transactions = append(info.Transactions, txns...)
		info.DebugLines = append(info.DebugLines, debugLines...)
	}

	// Post-process: determine debit/credit by comparing balance changes
	p.inferDebitCreditFromBalances(info.Transactions)

	return info, nil
}

// normalizeLine cleans up common PDF extraction artifacts.
func normalizeLine(line string) string {
	line = strings.ReplaceAll(line, "\u00A3", "£")
	line = strings.ReplaceAll(line, "\u200B", "")
	line = strings.ReplaceAll(line, "\u00A0", " ")
	return strings.TrimSpace(line)
}

func (p *HSBCParser) parseLines(lines []string) ([]models.Transaction, []models.DebugLine) {
	var transactions []models.Transaction
	var debugLines []models.DebugLine
	inTransactionSection := false

	for i := 0; i < len(lines); i++ {
		line := normalizeLine(lines[i])
		if line == "" {
			continue
		}

		hasDate := startsWithDate(line)
		hasTab := strings.Contains(line, "\t")
		tabParts := 0
		if hasTab {
			tabParts = len(strings.Split(line, "\t"))
		}

		dl := models.DebugLine{
			LineNum:  i + 1,
			HasDate:  hasDate,
			HasTab:   hasTab,
			TabParts: tabParts,
		}
		// Truncate long lines for debug display
		if len(line) > 120 {
			dl.Text = line[:120] + "..."
		} else {
			dl.Text = line
		}

		if containsTransactionHeader(line) {
			inTransactionSection = true
			dl.Result = "header"
			debugLines = append(debugLines, dl)
			continue
		}

		if !inTransactionSection && !hasDate {
			dl.Result = "skipped-pre-section"
			debugLines = append(debugLines, dl)
			continue
		}

		if hasDate {
			inTransactionSection = true
		}

		// Try tab-separated format first (from pdf.js client-side extraction)
		if hasTab {
			if txn, ok := p.tryTabSeparated(line); ok {
				txn.ParseMethod = "tab-separated"
				transactions = append(transactions, txn)
				dl.Result = "parsed"
				dl.Method = "tab-separated"
				debugLines = append(debugLines, dl)
				continue
			}
		}

		// Try strict text-date pattern (DD Mon YY) with double-space column separator
		if txn, ok := p.tryPattern(hsbcTxnPattern, line); ok {
			txn.ParseMethod = "strict-text-date"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "strict-text-date"
			debugLines = append(debugLines, dl)
			continue
		}

		// Try flexible text-date pattern (single-space separator)
		if txn, ok := p.tryPattern(hsbcTxnFlexible, line); ok {
			txn.ParseMethod = "flexible-text-date"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "flexible-text-date"
			debugLines = append(debugLines, dl)
			continue
		}

		// Try dash-date pattern (DD-Mon-YY)
		if txn, ok := p.tryPattern(hsbcDashDatePattern, line); ok {
			txn.ParseMethod = "dash-date"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "dash-date"
			debugLines = append(debugLines, dl)
			continue
		}

		// Try slash-date pattern (DD/MM/YYYY)
		if txn, ok := p.tryPattern(hsbcSlashDatePattern, line); ok {
			txn.ParseMethod = "slash-date"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "slash-date"
			debugLines = append(debugLines, dl)
			continue
		}

		// Try simple (date + description + one amount)
		if m := hsbcTxnSimple.FindStringSubmatch(line); m != nil {
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
			txn.ParseMethod = "simple"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "simple"
			debugLines = append(debugLines, dl)
			continue
		}

		// Try generic: line starts with date, has amounts somewhere at the end
		if txn, ok := p.tryGenericDateLine(line); ok {
			txn.ParseMethod = "generic-date-line"
			transactions = append(transactions, txn)
			dl.Result = "parsed"
			dl.Method = "generic-date-line"
			debugLines = append(debugLines, dl)
			continue
		}

		// Look-ahead: if this line has a date but no amounts were found,
		// peek at the next line. If it doesn't start with a date and contains
		// tab-separated amounts, join them and re-try (handles PDF line-split).
		if hasDate && i+1 < len(lines) {
			nextLine := normalizeLine(lines[i+1])
			if nextLine != "" && !startsWithDate(nextLine) {
				combined := line + "\t" + nextLine
				if txn, ok := p.tryTabSeparated(combined); ok {
					txn.ParseMethod = "tab-separated-joined"
					transactions = append(transactions, txn)
					dl.Result = "parsed"
					dl.Method = "tab-joined"
					dl.Text = dl.Text + " ⊕ " + nextLine
					debugLines = append(debugLines, dl)
					i++ // skip the next line since we consumed it
					continue
				}
			}
		}

		// Multi-line description continuation
		if len(transactions) > 0 && !hasDate && inTransactionSection {
			if !isSummaryLine(line) {
				last := &transactions[len(transactions)-1]
				cleaned := strings.ReplaceAll(line, "\t", " ")
				if !amountCellPattern.MatchString(strings.TrimSpace(cleaned)) {
					last.Description += " " + strings.TrimSpace(cleaned)
					dl.Result = "continuation"
					debugLines = append(debugLines, dl)
					continue
				}
			}
		}

		dl.Result = "unmatched"
		debugLines = append(debugLines, dl)
	}

	return transactions, debugLines
}

// tryTabSeparated handles tab-separated lines from pdf.js extraction.
// Strategy: find the date at the start, find amounts from the right side,
// everything in between is the description.
func (p *HSBCParser) tryTabSeparated(line string) (models.Transaction, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return models.Transaction{}, false
	}

	// Clean up parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// First part (or first part of first cell) should contain a date
	date := extractDate(parts[0])
	if date == "" {
		return models.Transaction{}, false
	}

	// Find where the date actually starts in the first cell
	// (handles stray characters like "A 30 Dec 25" where "A" is a PDF artifact)
	dateIdx := strings.Index(parts[0], date)
	if dateIdx < 0 {
		return models.Transaction{}, false
	}

	// Scan from the right to find amount cells
	var amounts []float64
	rightBoundary := len(parts)
	for i := len(parts) - 1; i >= 1; i-- {
		cell := strings.TrimSpace(parts[i])
		if cell == "" {
			continue // skip empty cells (empty column)
		}
		if m := amountCellPattern.FindStringSubmatch(cell); m != nil {
			amt, _ := parseAmount(m[1])
			amounts = append([]float64{amt}, amounts...) // prepend to keep order
			rightBoundary = i
		} else {
			break // stop at first non-amount cell
		}
	}

	if len(amounts) == 0 {
		return models.Transaction{}, false
	}

	// Build description from everything between date and amounts
	var descParts []string
	// Rest of first cell after the date (skip any chars before the date too)
	rest := strings.TrimSpace(parts[0][dateIdx+len(date):])
	if rest != "" {
		descParts = append(descParts, rest)
	}
	for i := 1; i < rightBoundary; i++ {
		cell := strings.TrimSpace(parts[i])
		// Skip empty cells and PDF artifacts (single dots, single chars)
		if cell == "" || cell == "." || cell == "-" || cell == "–" {
			continue
		}
		descParts = append(descParts, cell)
	}
	description := strings.Join(descParts, " ")

	txn := models.Transaction{
		Date:        date,
		Description: description,
	}

	// Assign amounts based on count
	switch len(amounts) {
	case 1:
		// Just a balance (e.g., "BALANCE BROUGHT FORWARD")
		txn.Balance = amounts[0]
		txn.Amount = 0
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	case 2:
		// One amount + balance
		txn.Amount = amounts[0]
		txn.Balance = amounts[1]
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	case 3:
		// paidOut + paidIn + balance
		txn.Balance = amounts[2]
		if amounts[0] > 0 && amounts[1] == 0 {
			txn.Amount = amounts[0]
			txn.Type = "DEBIT"
		} else if amounts[1] > 0 {
			txn.Amount = amounts[1]
			txn.Type = "CREDIT"
		} else {
			txn.Amount = amounts[0]
			txn.Type = "DEBIT"
		}
	default:
		// More than 3 amounts — take last as balance, second-to-last as amount
		txn.Balance = amounts[len(amounts)-1]
		txn.Amount = amounts[len(amounts)-2]
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	}

	return txn, true
}

// tryGenericDateLine handles lines that start with a date and end with amounts,
// regardless of separator style.
var trailingAmountsPattern = regexp.MustCompile(`[£\x{00A3}]?([\d,]+\.\d{2})`)

func (p *HSBCParser) tryGenericDateLine(line string) (models.Transaction, bool) {
	date := extractDate(line)
	if date == "" {
		return models.Transaction{}, false
	}

	// Find where the date actually starts in the line
	// (handles stray characters before the date, e.g., "A 30 Dec 25")
	dateIdx := strings.Index(line, date)
	if dateIdx < 0 {
		return models.Transaction{}, false
	}
	rest := strings.TrimSpace(line[dateIdx+len(date):])
	if rest == "" {
		return models.Transaction{}, false
	}

	// Find all amounts in the line
	allAmounts := trailingAmountsPattern.FindAllStringIndex(rest, -1)
	if len(allAmounts) == 0 {
		return models.Transaction{}, false
	}

	// The description is everything before the first amount
	firstAmountStart := allAmounts[0][0]
	description := strings.TrimSpace(rest[:firstAmountStart])
	if description == "" {
		return models.Transaction{}, false
	}

	// Extract all amounts
	amountMatches := trailingAmountsPattern.FindAllStringSubmatch(rest, -1)
	var amounts []float64
	for _, m := range amountMatches {
		amt, _ := parseAmount(m[1])
		amounts = append(amounts, amt)
	}

	txn := models.Transaction{
		Date:        date,
		Description: description,
	}

	switch len(amounts) {
	case 1:
		txn.Balance = amounts[0]
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	case 2:
		txn.Amount = amounts[0]
		txn.Balance = amounts[1]
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	case 3:
		txn.Balance = amounts[2]
		if amounts[0] > 0 && amounts[1] == 0 {
			txn.Amount = amounts[0]
			txn.Type = "DEBIT"
		} else if amounts[1] > 0 {
			txn.Amount = amounts[1]
			txn.Type = "CREDIT"
		} else {
			txn.Amount = amounts[0]
			txn.Type = "DEBIT"
		}
	default:
		txn.Balance = amounts[len(amounts)-1]
		txn.Amount = amounts[len(amounts)-2]
		if isDebitDescription(description) {
			txn.Type = "DEBIT"
		} else {
			txn.Type = "CREDIT"
		}
	}

	return txn, true
}

func (p *HSBCParser) tryPattern(pat *regexp.Regexp, line string) (models.Transaction, bool) {
	m := pat.FindStringSubmatch(line)
	if m == nil {
		return models.Transaction{}, false
	}

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

	return txn, true
}

// inferDebitCreditFromBalances uses balance progression to determine
// whether each transaction is a debit or credit. This is more reliable
// than keyword matching because it uses actual accounting math.
func (p *HSBCParser) inferDebitCreditFromBalances(txns []models.Transaction) {
	for i := 1; i < len(txns); i++ {
		prev := txns[i-1]
		curr := &txns[i]

		// Only infer if both have a balance and current has an amount
		if prev.Balance == 0 || curr.Balance == 0 || curr.Amount == 0 {
			continue
		}

		diff := curr.Balance - prev.Balance
		if diff < 0 {
			// Balance went down — this is a debit (money out)
			curr.Type = "DEBIT"
			// If no amount was parsed, use the balance difference
			if curr.Amount == 0 {
				curr.Amount = abs(diff)
			}
		} else if diff > 0 {
			// Balance went up — this is a credit (money in)
			curr.Type = "CREDIT"
			if curr.Amount == 0 {
				curr.Amount = diff
			}
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
