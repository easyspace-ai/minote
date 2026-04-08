package notex

import (
	"strings"
	"testing"
)

func TestStudioRawLooksLikeHTML(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"# Hello", false},
		{"### **Bold**", false},
		{"| a | b |\n|---|---|", false},
		{"<!DOCTYPE html><html>", true},
		{"<HTML lang=\"en\">", true},
		{"<p>Hello</p>", true},
		{"<div class=\"x\">\n<p>y</p>\n</div>", true},
	}
	for _, tc := range cases {
		if got := studioRawLooksLikeHTML(tc.in); got != tc.want {
			t.Errorf("studioRawLooksLikeHTML(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestStudioHTMLDocumentFromUserContent_MarkdownToPage(t *testing.T) {
	t.Parallel()
	md := "### 标题\n\n**粗体**\n\n| 列A | 列B |\n| --- | --- |\n| 1 | 2 |"
	out := studioHTMLDocumentFromUserContent("测试页", md)
	if !strings.Contains(out, "<h3") || !strings.Contains(out, "标题") {
		t.Fatalf("expected rendered h3, got snippet: %s", truncate(out, 400))
	}
	if !strings.Contains(out, "<strong>") && !strings.Contains(out, "<b>") {
		t.Fatalf("expected bold markup, got: %s", truncate(out, 500))
	}
	if !strings.Contains(out, "<table") {
		t.Fatalf("expected GFM table, got: %s", truncate(out, 600))
	}
	if strings.Contains(out, "###") {
		t.Fatalf("raw markdown heading leaked into HTML")
	}
}

func TestStudioHTMLDocumentFromUserContent_PreservesFullHTML(t *testing.T) {
	t.Parallel()
	doc := "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>X</title></head><body><p>ok</p></body></html>"
	out := studioHTMLDocumentFromUserContent("ignored", doc)
	if out != doc {
		t.Fatalf("full document should pass through unchanged")
	}
}

func TestStudioHTMLDocumentFromUserContent_HTMLFragmentWrap(t *testing.T) {
	t.Parallel()
	frag := "<main><h1>Hi</h1></main>"
	out := studioHTMLDocumentFromUserContent("T", frag)
	if !strings.Contains(out, "<main>") || !strings.Contains(out, "<title>T</title>") {
		t.Fatalf("expected wrapped fragment, got: %s", truncate(out, 300))
	}
}

func TestStripStudioOuterCodeFence(t *testing.T) {
	t.Parallel()
	in := "```html\n<p>x</p>\n```"
	got := stripStudioOuterCodeFence(in)
	if !strings.Contains(got, "<p>") || strings.Contains(got, "```") {
		t.Fatalf("got %q", got)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
