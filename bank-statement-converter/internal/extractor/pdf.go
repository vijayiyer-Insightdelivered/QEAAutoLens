package extractor

import (
	"fmt"
	"io"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

// ExtractText reads a PDF file and returns the text content of each page.
// It tries multiple extraction methods to handle different PDF encodings.
// If the structured PDF library fails, falls back to raw stream parsing
// and then to the external pdftotext command (poppler-utils).
func ExtractText(filePath string) ([]string, error) {
	// First, try the structured library (best layout preservation)
	pages, libErr := extractWithLibrary(filePath)
	if libErr == nil && isReadableText(pages) {
		return pages, nil
	}

	// Library failed or returned garbage — try raw stream extraction
	rawPages, rawErr := ExtractTextRaw(filePath)
	if rawErr == nil && isReadableText(rawPages) {
		return rawPages, nil
	}

	// Both Go methods failed — try external pdftotext (poppler-utils) as last resort
	popplerPages, popplerErr := extractWithPdftotext(filePath)
	if popplerErr == nil && isReadableText(popplerPages) {
		return popplerPages, nil
	}

	// All methods failed — never return garbage text
	if libErr != nil {
		return nil, fmt.Errorf("PDF text extraction failed: %v. The PDF may use custom fonts or be image-based/scanned. Text could not be decoded into readable content", libErr)
	}
	return nil, fmt.Errorf("no readable text could be extracted from PDF. The file may be image-based/scanned, or uses custom font encodings that cannot be decoded. Try opening the PDF in your browser, selecting all text (Ctrl+A), copying (Ctrl+C), and pasting into a text file")
}

// textQuality returns the ratio of basic ASCII readable characters (a-z, A-Z,
// 0-9, common punctuation, whitespace) to total characters. Returns 0.0-1.0.
// Uses a strict ASCII check — unicode.IsLetter() is too broad and matches
// accented characters that appear in garbage from identity-encoded fonts.
func textQuality(pages []string) float64 {
	total := 0
	readable := 0
	for _, page := range pages {
		for _, r := range page {
			total++
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || unicode.IsSpace(r) ||
				r == '.' || r == ',' || r == '-' || r == '/' || r == ':' ||
				r == ';' || r == '(' || r == ')' || r == '\'' || r == '"' ||
				r == '£' || r == '$' || r == '€' || r == '%' || r == '&' ||
				r == '@' || r == '#' || r == '!' || r == '?' || r == '+' ||
				r == '=' || r == '*' || r == '\t' {
				readable++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(readable) / float64(total)
}

// commonWords that appear in virtually all bank statements.
// If the extracted text contains none of these, it's likely garbage.
var commonWords = []string{
	"bank", "account", "balance", "date", "payment", "statement",
	"total", "amount", "credit", "debit", "transaction", "sort code",
	"money", "paid", "opening", "closing", "transfer", "direct",
	"number", "page", "period",
}

// containsCommonWords checks whether the text contains at least one word
// that would be expected in a bank statement.
func containsCommonWords(pages []string) bool {
	combined := strings.ToLower(strings.Join(pages, " "))
	for _, word := range commonWords {
		if strings.Contains(combined, word) {
			return true
		}
	}
	return false
}

// isReadableText checks that pages contain enough text, that it's actually
// readable (not binary garbage), AND that it contains recognizable words.
// Requires >50 chars, >60% readable ASCII characters, and at least one common word.
func isReadableText(pages []string) bool {
	if totalTextLen(pages) <= 50 {
		return false
	}
	if textQuality(pages) <= 0.6 {
		return false
	}
	// Final check: the text must contain at least one recognizable word
	return containsCommonWords(pages)
}

// IsReadableText is the exported version for use by other packages.
func IsReadableText(pages []string) bool {
	return isReadableText(pages)
}

// extractWithPdftotext uses the external pdftotext command from poppler-utils
// as a fallback for PDFs that the Go library cannot handle.
func extractWithPdftotext(filePath string) ([]string, error) {
	// Check if pdftotext is available
	_, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil, fmt.Errorf("pdftotext not available: %v", err)
	}

	// First, get the number of pages
	pageCountOut, err := exec.Command("pdfinfo", filePath).Output()
	numPages := 1
	if err == nil {
		for _, line := range strings.Split(string(pageCountOut), "\n") {
			if strings.HasPrefix(line, "Pages:") {
				n, parseErr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
				if parseErr == nil && n > 0 {
					numPages = n
				}
			}
		}
	}

	// Extract each page separately to preserve page boundaries
	var pages []string
	for i := 1; i <= numPages; i++ {
		pageStr := strconv.Itoa(i)
		out, err := exec.Command("pdftotext", "-layout", "-f", pageStr, "-l", pageStr, filePath, "-").Output()
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(out))
		if text != "" {
			pages = append(pages, text)
		}
	}

	if len(pages) == 0 {
		// Try whole document at once as fallback
		out, err := exec.Command("pdftotext", "-layout", filePath, "-").Output()
		if err != nil {
			return nil, fmt.Errorf("pdftotext failed: %v", err)
		}
		text := strings.TrimSpace(string(out))
		if text != "" {
			return []string{text}, nil
		}
		return nil, fmt.Errorf("pdftotext produced no output")
	}

	return pages, nil
}

// extractWithLibrary uses the ledongthuc/pdf library with multiple methods.
func extractWithLibrary(filePath string) (pages []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF library crashed: %v", r)
		}
	}()

	f, r, openErr := pdf.Open(filePath)
	if openErr != nil {
		return nil, openErr
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF has no pages")
	}

	// Method 1: Try GetTextByRow (best layout preservation)
	pages = extractByRow(r, numPages)
	if isReadableText(pages) {
		return pages, nil
	}

	// Method 2: Try Page.Content() with coordinate-based row reconstruction
	pages = extractByContent(r, numPages)
	if isReadableText(pages) {
		return pages, nil
	}

	// Method 3: Try Page.GetPlainText with font map
	pages = extractByPagePlainText(r, numPages)
	if isReadableText(pages) {
		return pages, nil
	}

	// Method 4: Try Reader.GetPlainText (different extraction path)
	plainText := extractByReaderPlainText(r)
	if isReadableText([]string{plainText}) {
		return []string{plainText}, nil
	}

	return pages, nil
}

// ExtractTextCombined reads a PDF and returns all text combined into one string.
func ExtractTextCombined(filePath string) (string, error) {
	pages, err := ExtractText(filePath)
	if err != nil {
		return "", err
	}
	return strings.Join(pages, "\n\n"), nil
}

// Method 1: GetTextByRow — best for well-structured PDFs
func extractByRow(r *pdf.Reader, numPages int) []string {
	var pages []string
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		rows, err := page.GetTextByRow()
		if err != nil {
			continue
		}
		var lines []string
		for _, row := range rows {
			var parts []string
			for _, word := range row.Content {
				parts = append(parts, word.S)
			}
			line := strings.Join(parts, " ")
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
		pages = append(pages, strings.Join(lines, "\n"))
	}
	return pages
}

// Method 2: Page.Content() — lower-level access to text objects.
// Groups text pieces by Y coordinate to reconstruct rows, then sorts by X.
func extractByContent(r *pdf.Reader, numPages int) []string {
	var pages []string
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content := page.Content()
		if len(content.Text) == 0 {
			continue
		}

		// Group text by Y coordinate (row), allowing small tolerance
		type textItem struct {
			x float64
			s string
		}
		rowMap := make(map[int][]textItem)
		for _, t := range content.Text {
			if strings.TrimSpace(t.S) == "" {
				continue
			}
			// Round Y to nearest integer to group into rows
			yKey := int(math.Round(t.Y))
			rowMap[yKey] = append(rowMap[yKey], textItem{x: t.X, s: t.S})
		}

		// Sort rows by Y (descending — PDF Y goes bottom-to-top)
		yKeys := make([]int, 0, len(rowMap))
		for y := range rowMap {
			yKeys = append(yKeys, y)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(yKeys)))

		var lines []string
		for _, y := range yKeys {
			items := rowMap[y]
			// Sort items in row by X coordinate (left to right)
			sort.Slice(items, func(a, b int) bool {
				return items[a].x < items[b].x
			})

			var parts []string
			var prevX float64
			for j, item := range items {
				if j > 0 && item.x-prevX > 15 {
					// Large gap between text items — insert extra space as column separator
					parts = append(parts, "  ")
				}
				parts = append(parts, item.s)
				prevX = item.x
			}
			line := strings.TrimSpace(strings.Join(parts, ""))
			if line != "" {
				lines = append(lines, line)
			}
		}
		pages = append(pages, strings.Join(lines, "\n"))
	}
	return pages
}

// Method 3: Page.GetPlainText with fonts
func extractByPagePlainText(r *pdf.Reader, numPages int) []string {
	var pages []string
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		// Build font map for the page
		fontNames := page.Fonts()
		fonts := make(map[string]*pdf.Font)
		for _, name := range fontNames {
			f := page.Font(name)
			fonts[name] = &f
		}

		text, err := page.GetPlainText(fonts)
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			pages = append(pages, text)
		}
	}
	return pages
}

// Method 4: Reader.GetPlainText — whole-document extraction
func extractByReaderPlainText(r *pdf.Reader) string {
	reader, err := r.GetPlainText()
	if err != nil {
		return ""
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func totalTextLen(pages []string) int {
	n := 0
	for _, p := range pages {
		n += len(strings.TrimSpace(p))
	}
	return n
}
