package langgraphcompat

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsPlausibleExtractedDocumentText returns false when s looks like binary or PDF stream garbage
// mis-decoded as UTF-8 (common with naive stream scraping). Used to avoid injecting乱码 into RAG/Studio.
func IsPlausibleExtractedDocumentText(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	s = strings.ToValidUTF8(s, "")
	if s == "" {
		return false
	}
	// Replacement runes usually mean invalid UTF-8 sequences in the original bytes.
	if strings.Count(s, "\uFFFD") > max(8, utf8.RuneCountInString(s)/200) {
		return false
	}

	var letterOrNumber, space, control, total int
	for _, r := range s {
		total++
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			space++
		case unicode.IsSpace(r):
			space++
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			letterOrNumber++
		case r < 0x20:
			control++
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			// punctuation common in markdown / CJK books
		default:
			if !unicode.IsPrint(r) {
				control++
			}
		}
	}
	if total == 0 {
		return false
	}
	if control*20 > total { // >5% control
		return false
	}
	// Need a minimum of real linguistic/numeric characters vs length (filters binary-ish noise).
	if letterOrNumber < 40 && utf8.RuneCountInString(s) > 300 {
		return false
	}
	// At least ~12% letters/digits for longer extracts; short notes can be smaller.
	runes := utf8.RuneCountInString(s)
	minRatio := 0.12
	if runes < 120 {
		minRatio = 0.08
	}
	if float64(letterOrNumber)/float64(runes) < minRatio {
		return false
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
