package langgraphcompat

import (
	"strings"
	"testing"
)

func TestIsPlausibleExtractedDocumentText(t *testing.T) {
	if !IsPlausibleExtractedDocumentText("这是一段用于测试的中文正文，包含足够的字母与汉字以便通过质量校验。") {
		t.Fatal("expected Chinese prose to pass")
	}
	if !IsPlausibleExtractedDocumentText("The quick brown fox jumps over the lazy dog. " + strings.Repeat("More words here. ", 40)) {
		t.Fatal("expected English prose to pass")
	}
	garbage := string([]byte{0xc3, 0x28, 0xa0, 0xff, 0xfe}) + "@@@###$$$%%%"
	for range 120 {
		garbage += "\x7f\x80\x81"
	}
	if IsPlausibleExtractedDocumentText(garbage) {
		t.Fatal("expected binary-ish string to fail")
	}
}
