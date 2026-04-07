package notex

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestBuildPPTXFromMarkdownOutline_ZipValid(t *testing.T) {
	b, err := buildPPTXFromMarkdownOutline("Demo", `# Demo

## First
- Point A
- Point B

## Second
- Only one`)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var hasCT, hasPres bool
	for _, f := range zr.File {
		switch f.Name {
		case "[Content_Types].xml":
			hasCT = true
		case "ppt/presentation.xml":
			hasPres = true
		}
	}
	if !hasCT || !hasPres {
		t.Fatalf("missing parts: ct=%v pres=%v", hasCT, hasPres)
	}
}

func TestParseStudioSlideOutline(t *testing.T) {
	s := parseStudioSlideOutline("T", "## A\n- x\n- y\n## B\n- z")
	if len(s) != 2 || s[0].Title != "A" || len(s[0].Bullets) != 2 || s[1].Bullets[0] != "z" {
		t.Fatalf("got %+v", s)
	}
}

func TestParseStudioSlideOutline_NumberedAndHeading3(t *testing.T) {
	s := parseStudioSlideOutline("Deck", "### First slide\n1. Alpha\n2. Beta\n### Second\n- Gamma")
	if len(s) != 2 {
		t.Fatalf("want 2 slides, got %d %+v", len(s), s)
	}
	if s[0].Title != "First slide" || len(s[0].Bullets) != 2 || s[0].Bullets[0] != "Alpha" || s[0].Bullets[1] != "Beta" {
		t.Fatalf("slide0 %+v", s[0])
	}
	if s[1].Title != "Second" || len(s[1].Bullets) != 1 || s[1].Bullets[0] != "Gamma" {
		t.Fatalf("slide1 %+v", s[1])
	}
}
