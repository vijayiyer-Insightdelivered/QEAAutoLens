package extractor

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText reads a PDF file and returns the text content of each page.
// It tries multiple extraction methods to handle different PDF encodings.
func ExtractText(filePath string) (pages []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF library crashed (recovered): %v", r)
		}
	}()

	f, r, openErr := pdf.Open(filePath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open PDF %q: %w", filePath, openErr)
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF %q has no pages", filePath)
	}

	// Method 1: Try GetTextByRow (best layout preservation)
	pages = extractByRow(r, numPages)
	if totalTextLen(pages) > 50 {
		return pages, nil
	}

	// Method 2: Try Page.Content() with coordinate-based row reconstruction
	pages = extractByContent(r, numPages)
	if totalTextLen(pages) > 50 {
		return pages, nil
	}

	// Method 3: Try Page.GetPlainText with font map
	pages = extractByPagePlainText(r, numPages)
	if totalTextLen(pages) > 50 {
		return pages, nil
	}

	// Method 4: Try Reader.GetPlainText (different extraction path)
	plainText := extractByReaderPlainText(r)
	if len(strings.TrimSpace(plainText)) > 50 {
		return []string{plainText}, nil
	}

	// Return whatever we got (may be empty)
	if totalTextLen(pages) > 0 {
		return pages, nil
	}
	if len(strings.TrimSpace(plainText)) > 0 {
		return []string{plainText}, nil
	}

	return nil, fmt.Errorf("PDF %q: no text could be extracted (the PDF may be image-based/scanned)", filePath)
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
