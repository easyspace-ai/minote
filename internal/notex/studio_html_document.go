package notex

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var studioGoldmark = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
)

// stripStudioOuterCodeFence removes a single leading ```lang fenced block if the whole payload is wrapped in it.
func stripStudioOuterCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return s
	}
	rest := strings.TrimPrefix(t, "```")
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i+1:]
	} else {
		return s
	}
	if j := strings.LastIndex(rest, "```"); j >= 0 {
		rest = strings.TrimSpace(rest[:j])
		return rest
	}
	return s
}

var (
	studioHTMLFirstLineRE    = regexp.MustCompile(`^</?[a-z][\w-]*(?:\s[\s\S]*?)?(?:/\s*>|>)$`)
	studioHTMLClosingTagRE   = regexp.MustCompile(`</[a-zA-Z][\w-]*\s*>`)
	studioHTMLSelfCloseEndRE = regexp.MustCompile(`/>\s*$`)
)

// studioRawLooksLikeHTML mirrors web StudioMaterialDialog.studioPayloadLooksLikeHtml, extended so single-line
// fragments like <p>...</p> count as HTML (otherwise they would be mis-parsed as Markdown).
func studioRawLooksLikeHTML(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "<!doctype") {
		return true
	}
	if strings.HasPrefix(lower, "<html") {
		return true
	}
	firstLine := t
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		firstLine = strings.TrimSpace(t[:i])
	}
	if len(firstLine) <= 120 && studioHTMLFirstLineRE.MatchString(firstLine) {
		return strings.Contains(t, "</") || strings.Contains(t, "/>")
	}
	if strings.HasPrefix(t, "<") && studioHTMLClosingTagRE.MatchString(t) {
		return true
	}
	if strings.HasPrefix(t, "<") && studioHTMLSelfCloseEndRE.MatchString(strings.TrimSpace(t)) {
		return true
	}
	return false
}

func studioMarkdownToHTMLFragment(md string) (string, error) {
	var buf bytes.Buffer
	if err := studioGoldmark.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const studioMarkdownArticleCSS = `
    :root { color-scheme: light dark; }
    body { margin: 0; font-family: 'PingFang SC','Microsoft YaHei',system-ui,sans-serif; line-height: 1.65; color: #1f2937; background: #fff; }
    @media (prefers-color-scheme: dark) {
      body { color: #e5e7eb; background: #0f172a; }
    }
    .studio-md-wrap { max-width: 52rem; margin: 0 auto; padding: 1.25rem 1.5rem 2.5rem; }
    .studio-md-wrap h1 { font-size: 1.75rem; margin: 1.25rem 0 0.75rem; }
    .studio-md-wrap h2 { font-size: 1.35rem; margin: 1.1rem 0 0.6rem; }
    .studio-md-wrap h3 { font-size: 1.15rem; margin: 1rem 0 0.5rem; }
    .studio-md-wrap p { margin: 0.5rem 0; }
    .studio-md-wrap ul, .studio-md-wrap ol { margin: 0.5rem 0; padding-left: 1.35rem; }
    .studio-md-wrap table { border-collapse: collapse; width: 100%; margin: 1rem 0; font-size: 0.9rem; overflow-x: auto; display: block; }
    .studio-md-wrap th, .studio-md-wrap td { border: 1px solid #d1d5db; padding: 0.35rem 0.5rem; text-align: left; }
    @media (prefers-color-scheme: dark) {
      .studio-md-wrap th, .studio-md-wrap td { border-color: #475569; }
    }
    .studio-md-wrap pre { overflow-x: auto; padding: 0.75rem; border-radius: 0.5rem; background: rgba(0,0,0,0.06); font-size: 0.85rem; }
    @media (prefers-color-scheme: dark) {
      .studio-md-wrap pre { background: rgba(255,255,255,0.08); }
    }
    .studio-md-wrap hr { border: none; border-top: 1px solid #e5e7eb; margin: 1.25rem 0; }
    @media (prefers-color-scheme: dark) {
      .studio-md-wrap hr { border-top-color: #334155; }
    }
    .studio-md-wrap blockquote { margin: 0.75rem 0; padding-left: 1rem; border-left: 3px solid #93c5fd; color: #4b5563; }
    @media (prefers-color-scheme: dark) {
      .studio-md-wrap blockquote { border-left-color: #3b82f6; color: #cbd5e1; }
    }
`

// studioHTMLDocumentFromUserContent builds a downloadable .html document. If the model returned Markdown
// instead of HTML, it is rendered with Goldmark (GFM) so browsers show formatted pages instead of a raw text wall.
func studioHTMLDocumentFromUserContent(title, raw string) string {
	t := strings.TrimSpace(stripStudioOuterCodeFence(raw))
	safeTitle := html.EscapeString(strings.TrimSpace(title))
	if safeTitle == "" {
		safeTitle = "Studio"
	}
	if t == "" {
		return "<!DOCTYPE html>\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>" +
			safeTitle + "</title></head><body><p></p></body></html>"
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
		return t
	}
	if studioRawLooksLikeHTML(t) {
		return "<!DOCTYPE html>\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>" +
			safeTitle + "</title></head>\n<body>\n" + t + "\n</body></html>"
	}
	frag, err := studioMarkdownToHTMLFragment(t)
	if err != nil || strings.TrimSpace(frag) == "" {
		frag = "<pre style=\"white-space:pre-wrap;font-family:system-ui,monospace\">" + html.EscapeString(t) + "</pre>"
	}
	return "<!DOCTYPE html>\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>" +
		safeTitle + "</title><style>" + studioMarkdownArticleCSS + "</style></head>\n<body>\n<div class=\"studio-md-wrap\">\n" + frag + "\n</div>\n</body></html>"
}
