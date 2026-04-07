package notex

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

//go:embed pptxdata/title_content_template.pptx
var pptxTitleContentTemplate []byte

type studioSlide struct {
	Title   string
	Bullets []string
}

var reH1 = regexp.MustCompile(`^#\s+(.+)\s*$`)
var reH2 = regexp.MustCompile(`^##\s+(.+)\s*$`)
var reH3Plus = regexp.MustCompile(`^#{3,6}\s+(.+)\s*$`)
var reNumberBullet = regexp.MustCompile(`^\d+\.\s+(.+)\s*$`)

func parseStudioSlideOutline(deckTitle string, md string) []studioSlide {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var slides []studioSlide
	var cur *studioSlide

	stripInlineMD := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "**", "")
		s = strings.ReplaceAll(s, "__", "")
		return strings.TrimSpace(s)
	}

	flush := func() {
		if cur == nil {
			return
		}
		t := stripInlineMD(cur.Title)
		var bs []string
		for _, b := range cur.Bullets {
			b = stripInlineMD(b)
			if b != "" {
				bs = append(bs, b)
			}
		}
		if t != "" || len(bs) > 0 {
			slides = append(slides, studioSlide{Title: t, Bullets: bs})
		}
		cur = nil
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if m := reH1.FindStringSubmatch(t); m != nil {
			flush()
			cur = &studioSlide{Title: stripInlineMD(m[1])}
			continue
		}
		if m := reH2.FindStringSubmatch(t); m != nil {
			flush()
			cur = &studioSlide{Title: stripInlineMD(m[1])}
			continue
		}
		if m := reH3Plus.FindStringSubmatch(t); m != nil {
			flush()
			cur = &studioSlide{Title: stripInlineMD(m[1])}
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			b := stripInlineMD(line[2:])
			if b != "" {
				if cur == nil {
					cur = &studioSlide{Title: stripInlineMD(deckTitle)}
				}
				cur.Bullets = append(cur.Bullets, b)
			}
			continue
		}
		if m := reNumberBullet.FindStringSubmatch(t); m != nil {
			b := stripInlineMD(m[1])
			if b != "" {
				if cur == nil {
					cur = &studioSlide{Title: stripInlineMD(deckTitle)}
				}
				cur.Bullets = append(cur.Bullets, b)
			}
			continue
		}
	}
	flush()

	if len(slides) == 0 {
		return nil
	}
	// Merge leading H1-only slide with next ## slide if the first has no bullets and second exists.
	if len(slides) >= 2 && len(slides[0].Bullets) == 0 && slides[0].Title != "" {
		slides[1].Title = slides[0].Title + " — " + slides[1].Title
		slides = slides[1:]
	}
	return slides
}

func stripDisallowedXMLChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x09, r == 0x0A, r == 0x0D:
			b.WriteRune(r)
		case r < 0x20:
			// XML 1.0 disallows most C0 controls
		case r == 0xFFFE || r == 0xFFFF:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func escapePPTXText(s string) string {
	s = strings.TrimSpace(stripDisallowedXMLChars(s))
	if s == "" {
		return " "
	}
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`'`, `&apos;`,
		`"`, `&quot;`,
	)
	return replacer.Replace(s)
}

// Matches placeholder runs in the embedded template (Keynote/PowerPoint pick up lang for fonts).
const pptxRunRPr = `<a:rPr lang="zh-CN" smtClean="0"/>`

func bulletParagraphsXML(bullets []string) string {
	if len(bullets) == 0 {
		return `<a:p><a:r>` + pptxRunRPr + `<a:t> </a:t></a:r><a:endParaRPr lang="zh-CN"/></a:p>`
	}
	var b strings.Builder
	for _, line := range bullets {
		x := escapePPTXText(line)
		b.WriteString(`<a:p><a:pPr marL="285750" indent="-285750"><a:buChar char="•"/></a:pPr><a:r>`)
		b.WriteString(pptxRunRPr)
		b.WriteString(`<a:t>`)
		b.WriteString(x)
		b.WriteString(`</a:t></a:r><a:endParaRPr lang="zh-CN"/></a:p>`)
	}
	return b.String()
}

// slideXMLFromChunk builds ppt/slides/slideN.xml body matching python-pptx title+content layout.
func slideXMLFromChunk(chunk studioSlide) []byte {
	title := escapePPTXText(chunk.Title)
	if title == "" {
		title = " "
	}
	body := bulletParagraphsXML(chunk.Bullets)
	s := `<?xml version='1.0' encoding='UTF-8' standalone='yes'?>` +
		`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/>` +
		`<p:sp><p:nvSpPr><p:cNvPr id="2" name="Title 1"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:spPr/>` +
		`<p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r>` + pptxRunRPr + `<a:t>` + title + `</a:t></a:r><a:endParaRPr lang="zh-CN"/></a:p></p:txBody></p:sp>` +
		`<p:sp><p:nvSpPr><p:cNvPr id="3" name="Content Placeholder 2"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr><p:ph idx="1"/></p:nvPr></p:nvSpPr><p:spPr/>` +
		`<p:txBody><a:bodyPr/><a:lstStyle/>` + body + `</p:txBody></p:sp>` +
		`</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`
	return []byte(s)
}

func nextRID(relsXML []byte) int {
	max := 0
	re := regexp.MustCompile(`Id="rId([0-9]+)"`)
	for _, m := range re.FindAllStringSubmatch(string(relsXML), -1) {
		n, _ := strconv.Atoi(m[1])
		if n > max {
			max = n
		}
	}
	return max + 1
}

func patchPPTXTemplateMultiSlide(template []byte, slides []studioSlide) ([]byte, error) {
	if len(slides) == 0 {
		return nil, fmt.Errorf("no slides")
	}
	zr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		files[f.Name] = b
	}

	slide1Rels, ok := files["ppt/slides/_rels/slide1.xml.rels"]
	if !ok {
		return nil, fmt.Errorf("template missing slide rels")
	}
	pres, ok := files["ppt/presentation.xml"]
	if !ok {
		return nil, fmt.Errorf("template missing presentation.xml")
	}
	presRels, ok := files["ppt/_rels/presentation.xml.rels"]
	if !ok {
		return nil, fmt.Errorf("template missing presentation.xml.rels")
	}
	ct, ok := files["[Content_Types].xml"]
	if !ok {
		return nil, fmt.Errorf("template missing [Content_Types].xml")
	}

	n := len(slides)
	firstSlideRID := nextRID(presRels)

	var sldIDs strings.Builder
	presRelsStr := string(presRels)
	reSlideRel := regexp.MustCompile(`<Relationship[^>]*Type="http://schemas\.openxmlformats\.org/officeDocument/2006/relationships/slide"[^>]*/>`)
	presRelsStr = reSlideRel.ReplaceAllString(presRelsStr, "")
	closeIdx := strings.LastIndex(presRelsStr, "</Relationships>")
	if closeIdx < 0 {
		return nil, fmt.Errorf("presentation.xml.rels: malformed")
	}
	var slideRelsAdded strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&slideRelsAdded, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, firstSlideRID+i, i+1)
	}
	files["ppt/_rels/presentation.xml.rels"] = []byte(presRelsStr[:closeIdx] + slideRelsAdded.String() + presRelsStr[closeIdx:])

	baseSldID := 256
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sldIDs, `<p:sldId id="%d" r:id="rId%d"/>`, baseSldID+i, firstSlideRID+i)
	}
	presStr := string(pres)
	reList := regexp.MustCompile(`<p:sldIdLst>[\s\S]*?</p:sldIdLst>`)
	if !reList.MatchString(presStr) {
		return nil, fmt.Errorf("presentation.xml: no sldIdLst")
	}
	presStr = reList.ReplaceAllString(presStr, `<p:sldIdLst>`+sldIDs.String()+`</p:sldIdLst>`)
	files["ppt/presentation.xml"] = []byte(presStr)

	ctStr := string(ct)
	var ctExtra strings.Builder
	reOverride := regexp.MustCompile(`<Override PartName="/ppt/slides/slide1.xml"[^>]*/>`)
	if !reOverride.MatchString(ctStr) {
		return nil, fmt.Errorf("[Content_Types].xml: no slide1 override")
	}
	for i := 1; i <= n; i++ {
		part := fmt.Sprintf(`/ppt/slides/slide%d.xml`, i)
		if i == 1 {
			continue
		}
		if strings.Contains(ctStr, part) {
			continue
		}
		fmt.Fprintf(&ctExtra, `<Override PartName="%s" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, part)
	}
	ctStr = strings.Replace(ctStr, `</Types>`, ctExtra.String()+`</Types>`, 1)
	files["[Content_Types].xml"] = []byte(ctStr)

	for i, sl := range slides {
		name := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
		files[name] = slideXMLFromChunk(sl)
		relName := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i+1)
		files[relName] = append([]byte(nil), slide1Rels...)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	// Deterministic order: match common OOXML expectations (Content_Types first helps some tools).
	ordered := reorderZipEntries(names)
	for _, name := range ordered {
		b := files[name]
		w, err := zw.Create(name)
		if err != nil {
			zw.Close()
			return nil, err
		}
		if _, err := w.Write(b); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func reorderZipEntries(names []string) []string {
	prio := func(s string) int {
		switch s {
		case "[Content_Types].xml":
			return 0
		case "_rels/.rels":
			return 1
		default:
			return 2
		}
	}
	// simple stable sort by (prio, name)
	type pair struct {
		p int
		n string
	}
	var ps []pair
	for _, n := range names {
		ps = append(ps, pair{prio(n), n})
	}
	for i := 0; i < len(ps); i++ {
		for j := i + 1; j < len(ps); j++ {
			if ps[j].p < ps[i].p || (ps[j].p == ps[i].p && ps[j].n < ps[i].n) {
				ps[i], ps[j] = ps[j], ps[i]
			}
		}
	}
	out := make([]string, len(ps))
	for i := range ps {
		out[i] = ps[i].n
	}
	return out
}

func buildPPTXFromMarkdownOutline(deckTitle string, md string) ([]byte, error) {
	md = strings.TrimSpace(md)
	slides := parseStudioSlideOutline(deckTitle, md)
	if len(slides) == 0 {
		t := strings.TrimSpace(deckTitle)
		if t == "" {
			t = "Presentation"
		}
		body := md
		if body == "" {
			body = " "
		}
		lines := strings.Split(body, "\n")
		var bullets []string
		for _, ln := range lines {
			x := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(ln, "**", ""), "__", ""))
			if x == "" {
				continue
			}
			bullets = append(bullets, x)
		}
		slides = []studioSlide{{Title: t, Bullets: bullets}}
	}
	return patchPPTXTemplateMultiSlide(pptxTitleContentTemplate, slides)
}
