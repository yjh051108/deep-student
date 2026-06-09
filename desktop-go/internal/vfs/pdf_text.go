package vfs

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const maxExtractedPdfTextBytes = 1_000_000

var (
	pdfPageObjectPattern = regexp.MustCompile(`/Type\s*/Page\b`)
	pdfCountPattern      = regexp.MustCompile(`/Count\s+([0-9]+)`)
)

func isPdfUpload(name string, mimeType string) bool {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	return lowerMime == "application/pdf" || strings.HasSuffix(lowerName, ".pdf")
}

func extractPdfTextLayer(data []byte) *string {
	if len(data) == 0 || !bytes.Contains(data[:minInt(len(data), 1024)], []byte("%PDF")) {
		return nil
	}

	parts := []string{}
	for _, stream := range pdfCandidateStreams(data) {
		parts = append(parts, scanPdfTextStrings(stream)...)
		if extractedTextLength(parts) > maxExtractedPdfTextBytes {
			break
		}
	}
	if len(parts) == 0 {
		parts = scanPdfTextStrings(data)
	}
	text := normalizePdfTextParts(parts)
	if text == "" {
		return nil
	}
	return &text
}

func detectPdfPageCount(data []byte) *int {
	if len(data) == 0 || !bytes.Contains(data[:minInt(len(data), 1024)], []byte("%PDF")) {
		return nil
	}

	if matches := pdfPageObjectPattern.FindAllIndex(data, -1); len(matches) > 0 {
		count := len(matches)
		return &count
	}

	if count := maxPdfPageTreeCount(data); count > 0 {
		return &count
	}
	return nil
}

func maxPdfPageTreeCount(data []byte) int {
	maxCount := 0
	cursor := 0
	for {
		index := bytes.Index(data[cursor:], []byte("/Type"))
		if index < 0 {
			break
		}
		index += cursor
		windowStart := index - 256
		if windowStart < 0 {
			windowStart = 0
		}
		windowEnd := index + 1024
		if windowEnd > len(data) {
			windowEnd = len(data)
		}
		window := data[windowStart:windowEnd]
		if bytes.Contains(window, []byte("/Pages")) {
			for _, match := range pdfCountPattern.FindAllSubmatch(window, -1) {
				if len(match) < 2 {
					continue
				}
				count, err := strconv.Atoi(string(match[1]))
				if err == nil && count > maxCount {
					maxCount = count
				}
			}
		}
		cursor = index + len("/Type")
	}
	return maxCount
}

func pdfCandidateStreams(data []byte) [][]byte {
	out := [][]byte{}
	cursor := 0
	for {
		streamIndex := bytes.Index(data[cursor:], []byte("stream"))
		if streamIndex < 0 {
			break
		}
		streamIndex += cursor
		contentStart := streamIndex + len("stream")
		if contentStart < len(data) && data[contentStart] == '\r' {
			contentStart++
			if contentStart < len(data) && data[contentStart] == '\n' {
				contentStart++
			}
		} else if contentStart < len(data) && data[contentStart] == '\n' {
			contentStart++
		}

		endOffset := bytes.Index(data[contentStart:], []byte("endstream"))
		if endOffset < 0 {
			break
		}
		contentEnd := contentStart + endOffset
		raw := bytes.TrimRight(data[contentStart:contentEnd], "\r\n")
		dictStart := streamIndex - 2048
		if dictStart < 0 {
			dictStart = 0
		}
		dict := data[dictStart:streamIndex]
		if bytes.Contains(dict, []byte("FlateDecode")) {
			if inflated, err := inflatePdfStream(raw); err == nil && len(inflated) > 0 {
				out = append(out, inflated)
			}
		} else if len(raw) > 0 {
			out = append(out, raw)
		}
		cursor = contentEnd + len("endstream")
	}
	return out
}

func inflatePdfStream(raw []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, maxExtractedPdfTextBytes+1))
}

func scanPdfTextStrings(content []byte) []string {
	parts := []string{}
	for i := 0; i < len(content); i++ {
		switch content[i] {
		case '(':
			raw, next, ok := readPdfLiteralString(content, i)
			if !ok {
				continue
			}
			if text := printablePdfString(raw); text != "" {
				parts = append(parts, text)
			}
			i = next - 1
		case '<':
			if i+1 < len(content) && content[i+1] == '<' {
				continue
			}
			next := bytes.IndexByte(content[i+1:], '>')
			if next < 0 {
				continue
			}
			rawHex := content[i+1 : i+1+next]
			if decoded, ok := decodePdfHexString(rawHex); ok {
				if text := printablePdfString(decoded); text != "" {
					parts = append(parts, text)
				}
			}
			i = i + next + 1
		}
	}
	return parts
}

func readPdfLiteralString(content []byte, start int) ([]byte, int, bool) {
	depth := 1
	out := []byte{}
	for i := start + 1; i < len(content); i++ {
		ch := content[i]
		if ch == '\\' {
			value, next := readPdfEscapedByte(content, i)
			if value != nil {
				out = append(out, *value)
			}
			i = next - 1
			continue
		}
		if ch == '(' {
			depth++
			out = append(out, ch)
			continue
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				return out, i + 1, true
			}
			out = append(out, ch)
			continue
		}
		out = append(out, ch)
	}
	return nil, len(content), false
}

func readPdfEscapedByte(content []byte, slash int) (*byte, int) {
	if slash+1 >= len(content) {
		return nil, slash + 1
	}
	next := content[slash+1]
	switch next {
	case 'n':
		value := byte('\n')
		return &value, slash + 2
	case 'r':
		value := byte('\r')
		return &value, slash + 2
	case 't':
		value := byte('\t')
		return &value, slash + 2
	case 'b':
		value := byte('\b')
		return &value, slash + 2
	case 'f':
		value := byte('\f')
		return &value, slash + 2
	case '(', ')', '\\':
		return &next, slash + 2
	case '\r':
		nextIndex := slash + 2
		if nextIndex < len(content) && content[nextIndex] == '\n' {
			nextIndex++
		}
		return nil, nextIndex
	case '\n':
		return nil, slash + 2
	}
	if next >= '0' && next <= '7' {
		value := 0
		end := slash + 1
		for end < len(content) && end < slash+4 && content[end] >= '0' && content[end] <= '7' {
			value = value*8 + int(content[end]-'0')
			end++
		}
		decoded := byte(value & 0xff)
		return &decoded, end
	}
	return &next, slash + 2
}

func decodePdfHexString(raw []byte) ([]byte, bool) {
	clean := make([]byte, 0, len(raw))
	for _, ch := range raw {
		if unicode.IsSpace(rune(ch)) {
			continue
		}
		clean = append(clean, ch)
	}
	if len(clean) == 0 {
		return nil, false
	}
	if len(clean)%2 == 1 {
		clean = append(clean, '0')
	}
	decoded := make([]byte, hex.DecodedLen(len(clean)))
	if _, err := hex.Decode(decoded, clean); err != nil {
		return nil, false
	}
	return decoded, true
}

func printablePdfString(raw []byte) string {
	text := decodePdfStringBytes(raw)
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || !hasReadableRune(text) {
		return ""
	}
	return text
}

func decodePdfStringBytes(raw []byte) string {
	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		return utf16BigEndianToString(raw[2:])
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		return utf16LittleEndianToString(raw[2:])
	}
	if looksLikeUTF16BE(raw) {
		return utf16BigEndianToString(raw)
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	runes := make([]rune, len(raw))
	for i, value := range raw {
		runes[i] = rune(value)
	}
	return string(runes)
}

func looksLikeUTF16BE(raw []byte) bool {
	if len(raw) < 4 || len(raw)%2 != 0 {
		return false
	}
	pairs := len(raw) / 2
	zeroHigh := 0
	for i := 0; i+1 < len(raw); i += 2 {
		if raw[i] == 0 && raw[i+1] != 0 {
			zeroHigh++
		}
	}
	return zeroHigh*2 >= pairs
}

func utf16BigEndianToString(raw []byte) string {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
	}
	return string(utf16.Decode(units))
}

func utf16LittleEndianToString(raw []byte) string {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	return string(utf16.Decode(units))
}

func normalizePdfTextParts(parts []string) string {
	out := []string{}
	seen := map[string]bool{}
	total := 0
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		if total+len(trimmed) > maxExtractedPdfTextBytes {
			break
		}
		out = append(out, trimmed)
		total += len(trimmed)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func hasReadableRune(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func extractedTextLength(parts []string) int {
	total := 0
	for _, part := range parts {
		total += len(part)
	}
	return total
}
