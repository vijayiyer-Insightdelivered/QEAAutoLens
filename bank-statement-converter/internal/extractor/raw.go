package extractor

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ExtractTextRaw is a fallback PDF text extractor that works directly with
// the raw PDF byte stream. It does not rely on the ledongthuc/pdf library,
// which crashes on some PDFs. Instead it:
//  1. Finds all stream/endstream blocks in the PDF
//  2. Decompresses FlateDecode (zlib) streams
//  3. Extracts text from PDF text operators (Tj, TJ, ', ")
//
// This handles PDFs that the structured parser chokes on.
func ExtractTextRaw(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	streams := extractStreams(data)
	if len(streams) == 0 {
		return nil, nil
	}

	var allText []string
	for _, stream := range streams {
		// Try to decompress (most PDF streams are FlateDecode/zlib)
		decompressed := tryDecompress(stream)

		// Extract text from the content stream
		text := extractTextFromStream(decompressed)
		if text != "" {
			allText = append(allText, text)
		}
	}

	if len(allText) == 0 {
		return nil, nil
	}

	// Merge streams that belong to the same page
	// (PDFs often have multiple streams per page)
	merged := mergePageText(allText)
	return merged, nil
}

// extractStreams finds all stream...endstream blocks in the PDF.
func extractStreams(data []byte) [][]byte {
	var streams [][]byte
	streamMarker := []byte("stream")
	endMarker := []byte("endstream")

	offset := 0
	for offset < len(data) {
		// Find "stream" keyword
		idx := bytes.Index(data[offset:], streamMarker)
		if idx < 0 {
			break
		}
		start := offset + idx + len(streamMarker)

		// Skip \r\n or \n after "stream"
		if start < len(data) && data[start] == '\r' {
			start++
		}
		if start < len(data) && data[start] == '\n' {
			start++
		}

		// Find "endstream"
		endIdx := bytes.Index(data[start:], endMarker)
		if endIdx < 0 {
			break
		}

		streamData := data[start : start+endIdx]
		if len(streamData) > 0 {
			streams = append(streams, streamData)
		}
		offset = start + endIdx + len(endMarker)
	}
	return streams
}

// tryDecompress attempts zlib decompression; returns original data if it fails.
func tryDecompress(data []byte) []byte {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return data
	}
	defer r.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		return data
	}
	return out
}

// PDF text operator patterns
var (
	// Matches text in parentheses for Tj operator: (text) Tj
	tjPattern = regexp.MustCompile(`\(([^)]*)\)\s*Tj`)
	// Matches text arrays for TJ operator: [(text) 123 (text)] TJ
	tjArrayPattern = regexp.MustCompile(`\[([^\]]*)\]\s*TJ`)
	// Matches individual strings within TJ arrays
	tjArrayStringPattern = regexp.MustCompile(`\(([^)]*)\)`)
	// Matches ' operator (move to next line and show text): (text) '
	tickPattern = regexp.MustCompile(`\(([^)]*)\)\s*'`)
)

// extractTextFromStream parses PDF content stream and extracts text.
func extractTextFromStream(data []byte) string {
	content := string(data)

	// Check if this looks like a content stream with text operators
	if !strings.Contains(content, "Tj") && !strings.Contains(content, "TJ") &&
		!strings.Contains(content, "BT") {
		return ""
	}

	var parts []string

	// Extract Tj strings
	for _, m := range tjPattern.FindAllStringSubmatch(content, -1) {
		text := decodePDFString(m[1])
		if text != "" {
			parts = append(parts, text)
		}
	}

	// Extract TJ array strings
	for _, m := range tjArrayPattern.FindAllStringSubmatch(content, -1) {
		arrayContent := m[1]
		for _, sm := range tjArrayStringPattern.FindAllStringSubmatch(arrayContent, -1) {
			text := decodePDFString(sm[1])
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	// Extract ' operator strings
	for _, m := range tickPattern.FindAllStringSubmatch(content, -1) {
		text := decodePDFString(m[1])
		if text != "" {
			parts = append(parts, text)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Join parts, using newline where there's a clear line break
	result := strings.Join(parts, " ")
	return result
}

// decodePDFString handles basic PDF string escapes.
func decodePDFString(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case '(':
				buf.WriteByte('(')
			case ')':
				buf.WriteByte(')')
			case '\\':
				buf.WriteByte('\\')
			default:
				// Octal escape: \ddd
				if s[i] >= '0' && s[i] <= '7' {
					val := int(s[i] - '0')
					for j := 1; j < 3 && i+j < len(s) && s[i+j] >= '0' && s[i+j] <= '7'; j++ {
						val = val*8 + int(s[i+j]-'0')
						i++
					}
					if val > 0 && val < 256 {
						buf.WriteByte(byte(val))
					}
				} else {
					buf.WriteByte(s[i])
				}
			}
		} else {
			buf.WriteByte(s[i])
		}
		i++
	}

	result := buf.String()
	// Filter out non-printable characters but keep common ones
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		return -1
	}, result)
	return strings.TrimSpace(cleaned)
}

// mergePageText attempts to group extracted text into logical pages.
// Content streams that produce short text are often metadata;
// longer ones are actual page content.
func mergePageText(texts []string) []string {
	var pages []string
	var current strings.Builder

	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		// If this chunk has enough content to be a page section, include it
		if len(t) > 20 {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(t)
		}
	}

	if current.Len() > 0 {
		pages = append(pages, current.String())
	}

	// If nothing passed the length filter, include everything
	if len(pages) == 0 {
		var all strings.Builder
		for _, t := range texts {
			t = strings.TrimSpace(t)
			if t != "" {
				if all.Len() > 0 {
					all.WriteString("\n")
				}
				all.WriteString(t)
			}
		}
		if all.Len() > 0 {
			pages = append(pages, all.String())
		}
	}

	return pages
}
