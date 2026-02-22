package extractor

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ExtractTextRaw is a fallback PDF text extractor that works directly with
// the raw PDF byte stream. It does not rely on the ledongthuc/pdf library.
//
// It handles PDFs with custom font encodings (CIDFont/Type0) by:
//  1. Finding all ToUnicode CMap streams and building character mappings
//  2. Finding content streams with text operators (Tj, TJ)
//  3. Decoding both literal strings (...) and hex strings <...>
//  4. Applying CMap translations to produce readable Unicode text
func ExtractTextRaw(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	streams := extractStreams(data)
	if len(streams) == 0 {
		return nil, nil
	}

	// Step 1: Find and parse all ToUnicode CMap tables
	cmaps := FindCMaps(data)
	var cmap *CMap
	if len(cmaps) > 0 {
		cmap = MergeCMaps(cmaps)
	}

	// Step 2: Extract text from content streams
	var allText []string
	for _, stream := range streams {
		decompressed := tryDecompress(stream)
		text := extractTextFromStream(decompressed, cmap)
		if text != "" {
			allText = append(allText, text)
		}
	}

	if len(allText) == 0 {
		return nil, nil
	}

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

// Patterns for PDF text operators
var (
	// Matches hex strings for Tj: <hex> Tj
	hexTjPattern = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*Tj`)
	// Matches literal strings for Tj: (text) Tj
	litTjPattern = regexp.MustCompile(`\(([^)]*)\)\s*Tj`)
	// Matches TJ arrays: [...] TJ
	tjArrayPattern = regexp.MustCompile(`\[([^\]]*)\]\s*TJ`)
	// Matches hex strings within TJ arrays
	hexInArrayRe = regexp.MustCompile(`<([0-9A-Fa-f]+)>`)
	// Matches literal strings within TJ arrays
	litInArrayRe = regexp.MustCompile(`\(([^)]*)\)`)
	// Matches ' operator
	tickPattern = regexp.MustCompile(`\(([^)]*)\)\s*'`)
	// Matches Td/TD operators for line detection (text positioning)
	tdPattern = regexp.MustCompile(`([\d.\-]+)\s+([\d.\-]+)\s+T[dD]`)
)

// extractTextFromStream parses a PDF content stream and extracts text.
func extractTextFromStream(data []byte, cmap *CMap) string {
	content := string(data)

	// Check if this is a content stream with text operators
	if !strings.Contains(content, "Tj") && !strings.Contains(content, "TJ") &&
		!strings.Contains(content, "BT") {
		return ""
	}

	// Process the stream sequentially to preserve text order and detect line breaks
	// We walk through BT...ET blocks and track text position operators
	var lines []string
	var currentLine strings.Builder

	// Split into BT...ET text blocks
	btBlocks := splitBTBlocks(content)
	for _, block := range btBlocks {
		blockLines := processTextBlock(block, cmap)
		lines = append(lines, blockLines...)
	}

	// If no BT blocks found, try global extraction
	if len(lines) == 0 {
		text := extractAllText(content, cmap)
		if text != "" {
			lines = append(lines, text)
		}
	}

	_ = currentLine // used in processTextBlock
	result := strings.Join(lines, "\n")
	return strings.TrimSpace(result)
}

// splitBTBlocks extracts content between BT and ET operators.
func splitBTBlocks(content string) []string {
	var blocks []string
	remaining := content
	for {
		btIdx := strings.Index(remaining, "BT")
		if btIdx < 0 {
			break
		}
		etIdx := strings.Index(remaining[btIdx:], "ET")
		if etIdx < 0 {
			break
		}
		block := remaining[btIdx : btIdx+etIdx+2]
		blocks = append(blocks, block)
		remaining = remaining[btIdx+etIdx+2:]
	}
	return blocks
}

// processTextBlock extracts lines of text from a BT...ET block.
func processTextBlock(block string, cmap *CMap) []string {
	var lines []string
	var currentLine strings.Builder

	// Process line by line within the block
	ops := strings.Split(block, "\n")
	for _, op := range ops {
		op = strings.TrimSpace(op)

		// Check for text positioning that implies a new line
		// Td/TD with negative Y offset means new line
		if tdPattern.MatchString(op) {
			if currentLine.Len() > 0 {
				line := strings.TrimSpace(currentLine.String())
				if line != "" {
					lines = append(lines, line)
				}
				currentLine.Reset()
			}
		}

		// T* operator means new line
		if op == "T*" {
			if currentLine.Len() > 0 {
				line := strings.TrimSpace(currentLine.String())
				if line != "" {
					lines = append(lines, line)
				}
				currentLine.Reset()
			}
		}

		// Extract text from Tj with hex strings
		for _, m := range hexTjPattern.FindAllStringSubmatch(op, -1) {
			text := decodeHexString(m[1], cmap)
			currentLine.WriteString(text)
		}

		// Extract text from Tj with literal strings
		for _, m := range litTjPattern.FindAllStringSubmatch(op, -1) {
			text := decodeLiteralString(m[1], cmap)
			currentLine.WriteString(text)
		}

		// Extract text from TJ arrays
		for _, m := range tjArrayPattern.FindAllStringSubmatch(op, -1) {
			text := decodeTJArray(m[1], cmap)
			currentLine.WriteString(text)
		}

		// Extract text from ' operator
		for _, m := range tickPattern.FindAllStringSubmatch(op, -1) {
			if currentLine.Len() > 0 {
				line := strings.TrimSpace(currentLine.String())
				if line != "" {
					lines = append(lines, line)
				}
				currentLine.Reset()
			}
			text := decodeLiteralString(m[1], cmap)
			currentLine.WriteString(text)
		}
	}

	if currentLine.Len() > 0 {
		line := strings.TrimSpace(currentLine.String())
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// extractAllText extracts all text from content without BT/ET block structure.
func extractAllText(content string, cmap *CMap) string {
	var parts []string

	for _, m := range hexTjPattern.FindAllStringSubmatch(content, -1) {
		text := decodeHexString(m[1], cmap)
		if text != "" {
			parts = append(parts, text)
		}
	}
	for _, m := range litTjPattern.FindAllStringSubmatch(content, -1) {
		text := decodeLiteralString(m[1], cmap)
		if text != "" {
			parts = append(parts, text)
		}
	}
	for _, m := range tjArrayPattern.FindAllStringSubmatch(content, -1) {
		text := decodeTJArray(m[1], cmap)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, " ")
}

// decodeHexString decodes a hex-encoded PDF string using CMap if available.
func decodeHexString(hexStr string, cmap *CMap) string {
	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return ""
	}

	// Try CMap decoding first
	if cmap != nil && len(cmap.charMap) > 0 {
		result := cmap.Decode(raw)
		if result != "" {
			return result
		}
	}

	// Fallback: try as direct UTF-16BE
	if len(raw)%2 == 0 && len(raw) >= 2 {
		var result strings.Builder
		for i := 0; i+1 < len(raw); i += 2 {
			cp := rune(raw[i])<<8 | rune(raw[i+1])
			if unicode.IsPrint(cp) || cp == ' ' {
				result.WriteRune(cp)
			}
		}
		if result.Len() > 0 {
			return result.String()
		}
	}

	// Last resort: treat as ASCII
	return cleanString(string(raw))
}

// decodeLiteralString decodes a literal PDF string using CMap if available.
func decodeLiteralString(s string, cmap *CMap) string {
	decoded := decodePDFEscapes(s)

	// Try CMap decoding
	if cmap != nil && len(cmap.charMap) > 0 {
		result := cmap.Decode([]byte(decoded))
		if result != "" && isPrintable(result) {
			return result
		}
	}

	return cleanString(decoded)
}

// decodeTJArray decodes a TJ array, which contains a mix of strings and numbers.
func decodeTJArray(arrayContent string, cmap *CMap) string {
	var parts []string

	// Extract hex strings
	hexMatches := hexInArrayRe.FindAllStringSubmatchIndex(arrayContent, -1)
	litMatches := litInArrayRe.FindAllStringSubmatchIndex(arrayContent, -1)

	// Combine and sort by position
	type match struct {
		pos    int
		isHex  bool
		groups []string
	}
	var all []match

	for _, idx := range hexMatches {
		all = append(all, match{
			pos:   idx[0],
			isHex: true,
			groups: []string{
				arrayContent[idx[0]:idx[1]],
				arrayContent[idx[2]:idx[3]],
			},
		})
	}
	for _, idx := range litMatches {
		all = append(all, match{
			pos:   idx[0],
			isHex: false,
			groups: []string{
				arrayContent[idx[0]:idx[1]],
				arrayContent[idx[2]:idx[3]],
			},
		})
	}

	// Sort by position
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].pos < all[j-1].pos; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}

	for _, m := range all {
		var text string
		if m.isHex {
			text = decodeHexString(m.groups[1], cmap)
		} else {
			text = decodeLiteralString(m.groups[1], cmap)
		}
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "")
}

// decodePDFEscapes handles basic PDF string escape sequences.
func decodePDFEscapes(s string) string {
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
				if s[i] >= '0' && s[i] <= '7' {
					val := int(s[i] - '0')
					for j := 1; j < 3 && i+j < len(s) && s[i+j] >= '0' && s[i+j] <= '7'; j++ {
						val = val*8 + int(s[i+j]-'0')
						i++
					}
					if val >= 0 && val < 256 {
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
	return buf.String()
}

// cleanString removes non-printable characters.
func cleanString(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		return -1
	}, s))
}

// isPrintable checks if a string contains mostly printable characters.
func isPrintable(s string) bool {
	if len(s) == 0 {
		return false
	}
	printable := 0
	for _, r := range s {
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			printable++
		}
	}
	return float64(printable)/float64(len([]rune(s))) > 0.5
}

// mergePageText groups extracted text into logical pages.
func mergePageText(texts []string) []string {
	var pages []string
	var current strings.Builder

	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) > 10 {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(t)
		}
	}

	if current.Len() > 0 {
		pages = append(pages, current.String())
	}

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
