package extractor

import (
	"encoding/hex"
	"regexp"
	"strings"
	"unicode/utf16"
)

// CMap represents a character code to Unicode mapping table.
// PDF fonts use CMap tables (especially ToUnicode) to map glyph codes
// to Unicode code points.
type CMap struct {
	// charMap maps hex-encoded character codes to Unicode strings
	charMap map[string]string
}

// NewCMap creates a new empty CMap.
func NewCMap() *CMap {
	return &CMap{charMap: make(map[string]string)}
}

var (
	// Matches beginbfchar ... endbfchar blocks
	bfCharBlockRe = regexp.MustCompile(`(?s)beginbfchar\s*(.*?)\s*endbfchar`)
	// Matches beginbfrange ... endbfrange blocks
	bfRangeBlockRe = regexp.MustCompile(`(?s)beginbfrange\s*(.*?)\s*endbfrange`)
	// Matches hex tokens like <00A3>
	hexTokenRe = regexp.MustCompile(`<([0-9A-Fa-f]+)>`)
)

// ParseCMap parses a CMap/ToUnicode stream content into a CMap structure.
func ParseCMap(content string) *CMap {
	cm := NewCMap()

	// Parse beginbfchar ... endbfchar blocks
	// Format: <srcCode> <unicodeValue>
	for _, block := range bfCharBlockRe.FindAllStringSubmatch(content, -1) {
		tokens := hexTokenRe.FindAllStringSubmatch(block[1], -1)
		for i := 0; i+1 < len(tokens); i += 2 {
			srcHex := strings.ToUpper(tokens[i][1])
			dstHex := tokens[i+1][1]
			uni := hexToUnicode(dstHex)
			if uni != "" {
				cm.charMap[srcHex] = uni
			}
		}
	}

	// Parse beginbfrange ... endbfrange blocks
	// Format: <startCode> <endCode> <startUnicode>
	// or:     <startCode> <endCode> [<uni1> <uni2> ...]
	for _, block := range bfRangeBlockRe.FindAllStringSubmatch(content, -1) {
		lines := strings.Split(block[1], "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check if this is an array mapping: <start> <end> [<u1> <u2> ...]
			if strings.Contains(line, "[") {
				parseRangeArray(cm, line)
				continue
			}

			tokens := hexTokenRe.FindAllStringSubmatch(line, -1)
			if len(tokens) < 3 {
				continue
			}

			startHex := tokens[0][1]
			endHex := tokens[1][1]
			dstHex := tokens[2][1]

			startCode := hexToInt(startHex)
			endCode := hexToInt(endHex)
			dstCode := hexToInt(dstHex)

			if startCode < 0 || endCode < 0 || dstCode < 0 {
				continue
			}

			hexLen := len(startHex)
			for code := startCode; code <= endCode; code++ {
				srcKey := intToHex(code, hexLen)
				uni := hexToUnicode(intToHex(dstCode+(code-startCode), len(dstHex)))
				if uni != "" {
					cm.charMap[srcKey] = uni
				}
			}
		}
	}

	return cm
}

// parseRangeArray handles: <start> <end> [<u1> <u2> ...]
func parseRangeArray(cm *CMap, line string) {
	// Split at '['
	bracketIdx := strings.Index(line, "[")
	if bracketIdx < 0 {
		return
	}
	before := line[:bracketIdx]
	after := line[bracketIdx:]

	tokens := hexTokenRe.FindAllStringSubmatch(before, -1)
	if len(tokens) < 2 {
		return
	}

	startHex := tokens[0][1]
	startCode := hexToInt(startHex)
	hexLen := len(startHex)

	// Extract unicode values from array
	uniTokens := hexTokenRe.FindAllStringSubmatch(after, -1)
	for i, ut := range uniTokens {
		code := startCode + i
		srcKey := intToHex(code, hexLen)
		uni := hexToUnicode(ut[1])
		if uni != "" {
			cm.charMap[srcKey] = uni
		}
	}
}

// Decode takes raw bytes from a PDF text string and returns the Unicode text
// using this CMap's mappings.
func (cm *CMap) Decode(raw []byte) string {
	if len(cm.charMap) == 0 {
		return ""
	}

	// Determine code length from the CMap keys (usually 2 or 4 hex chars = 1 or 2 bytes)
	codeByteLen := 1
	for k := range cm.charMap {
		codeByteLen = len(k) / 2
		break
	}
	if codeByteLen < 1 {
		codeByteLen = 1
	}

	var result strings.Builder
	for i := 0; i <= len(raw)-codeByteLen; i += codeByteLen {
		chunk := raw[i : i+codeByteLen]
		key := strings.ToUpper(hex.EncodeToString(chunk))
		if uni, ok := cm.charMap[key]; ok {
			result.WriteString(uni)
		} else {
			// Try single-byte lookup as fallback
			if codeByteLen > 1 {
				key1 := strings.ToUpper(hex.EncodeToString(chunk[:1]))
				if uni1, ok := cm.charMap[key1]; ok {
					result.WriteString(uni1)
					// Rewind: process remaining bytes
					i -= (codeByteLen - 1)
					continue
				}
			}
			// If the byte is printable ASCII, use it directly
			if codeByteLen == 1 && chunk[0] >= 32 && chunk[0] < 127 {
				result.WriteByte(chunk[0])
			}
		}
	}
	return result.String()
}

// DecodeString decodes a PDF string literal using the CMap.
func (cm *CMap) DecodeString(s string) string {
	return cm.Decode([]byte(s))
}

// hexToInt converts a hex string to an integer.
func hexToInt(h string) int {
	val := 0
	for _, c := range strings.ToUpper(h) {
		val <<= 4
		if c >= '0' && c <= '9' {
			val += int(c - '0')
		} else if c >= 'A' && c <= 'F' {
			val += int(c-'A') + 10
		} else {
			return -1
		}
	}
	return val
}

// intToHex converts an integer to a zero-padded uppercase hex string.
func intToHex(val, hexLen int) string {
	h := strings.ToUpper(hex.EncodeToString([]byte{byte(val >> 8), byte(val)}))
	// Trim or pad to match expected length
	if len(h) > hexLen {
		h = h[len(h)-hexLen:]
	}
	for len(h) < hexLen {
		h = "0" + h
	}
	return h
}

// hexToUnicode converts a hex-encoded Unicode value to a Go string.
// Handles both BMP (4 hex chars) and supplementary (8 hex chars / surrogate pairs).
func hexToUnicode(h string) string {
	// Pad to even length
	if len(h)%2 != 0 {
		h = "0" + h
	}

	data, err := hex.DecodeString(h)
	if err != nil {
		return ""
	}

	// Interpret as UTF-16BE
	if len(data) == 2 {
		cp := uint16(data[0])<<8 | uint16(data[1])
		return string(rune(cp))
	}

	if len(data) == 4 {
		// Could be a surrogate pair
		hi := uint16(data[0])<<8 | uint16(data[1])
		lo := uint16(data[2])<<8 | uint16(data[3])
		if hi >= 0xD800 && hi <= 0xDBFF && lo >= 0xDC00 && lo <= 0xDFFF {
			r := utf16.DecodeRune(rune(hi), rune(lo))
			return string(r)
		}
		// Two separate characters
		return string(rune(hi)) + string(rune(lo))
	}

	// For longer sequences, decode as series of UTF-16BE code units
	var result strings.Builder
	for i := 0; i+1 < len(data); i += 2 {
		cp := uint16(data[i])<<8 | uint16(data[i+1])
		result.WriteRune(rune(cp))
	}
	return result.String()
}

// FindCMaps searches the raw PDF bytes for all ToUnicode CMap streams.
func FindCMaps(data []byte) []*CMap {
	var cmaps []*CMap

	streams := extractStreams(data)
	for _, stream := range streams {
		decompressed := tryDecompress(stream)
		content := string(decompressed)

		// Check if this stream contains CMap data
		if strings.Contains(content, "beginbfchar") || strings.Contains(content, "beginbfrange") {
			cm := ParseCMap(content)
			if len(cm.charMap) > 0 {
				cmaps = append(cmaps, cm)
			}
		}
	}

	return cmaps
}

// MergeCMaps combines multiple CMaps into a single one.
func MergeCMaps(cmaps []*CMap) *CMap {
	merged := NewCMap()
	for _, cm := range cmaps {
		for k, v := range cm.charMap {
			merged.charMap[k] = v
		}
	}
	return merged
}
