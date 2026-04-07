package notex

// PPTX Generator V2 - 基于主题和样式的专业 PPT 生成
// 参考 PPT-as-code 设计理念，支持多种主题、布局和视觉效果

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

// PPTTheme 定义 PPT 主题配置
type PPTTheme struct {
	Name        string
	Primary     string // 主色
	Accent      string // 强调色
	Background  string // 背景色
	Text        string // 文字色
	TextLight   string // 浅色文字（用于深色背景）
	FontHeading string // 标题字体
	FontBody    string // 正文字体
}

// 预定义主题（与 YAML 配置对应）
var PPTThemes = map[string]PPTTheme{
	"professional": {
		Name:        "专业商务",
		Primary:     "1e3a5f",
		Accent:      "2563eb",
		Background:  "ffffff",
		Text:        "1f2937",
		TextLight:   "ffffff",
		FontHeading: "微软雅黑",
		FontBody:    "微软雅黑",
	},
	"minimal": {
		Name:        "极简风格",
		Primary:     "111827",
		Accent:      "374151",
		Background:  "f9fafb",
		Text:        "111827",
		TextLight:   "ffffff",
		FontHeading: "苹方",
		FontBody:    "苹方",
	},
	"vibrant": {
		Name:        "活力创意",
		Primary:     "7c3aed",
		Accent:      "06b6d4",
		Background:  "ffffff",
		Text:        "1f2937",
		TextLight:   "ffffff",
		FontHeading: "思源黑体",
		FontBody:    "思源黑体",
	},
	"tech": {
		Name:        "科技感",
		Primary:     "00d2ff",
		Accent:      "00ff88",
		Background:  "0a0a1a",
		Text:        "e2e8f0",
		TextLight:   "ffffff",
		FontHeading: "JetBrains Mono",
		FontBody:    "Inter",
	},
	"education": {
		Name:        "教育培训",
		Primary:     "d97706",
		Accent:      "059669",
		Background:  "fffbeb",
		Text:        "78350f",
		TextLight:   "ffffff",
		FontHeading: "楷体",
		FontBody:    "微软雅黑",
	},
}

// SlideLayout 幻灯片布局类型
type SlideLayout string

const (
	LayoutCover      SlideLayout = "cover"
	LayoutTOC        SlideLayout = "toc"
	LayoutSection    SlideLayout = "section"
	LayoutContent    SlideLayout = "content"
	LayoutTwoColumn  SlideLayout = "two_col"
	LayoutImageText  SlideLayout = "image_text"
	LayoutData       SlideLayout = "data"
	LayoutQuote      SlideLayout = "quote"
	LayoutSummary    SlideLayout = "summary"
	LayoutEnd        SlideLayout = "end"
)

// SlideContent 单张幻灯片内容
type SlideContent struct {
	Type     SlideLayout
	Title    string
	Subtitle string
	Bullets  []string
	Notes    string
	LayoutHint string
}

// BuildThemedPPTX 根据主题和布局生成 PPTX
func BuildThemedPPTX(title string, slides []SlideContent, themeName string) ([]byte, error) {
	theme, ok := PPTThemes[themeName]
	if !ok {
		theme = PPTThemes["professional"]
	}

	// 生成主题色的 XML
	themeXML := generateThemeXML(theme)

	// 生成演示文稿结构
	presXML := generatePresentationXML(title, slides, theme)

	// 生成各幻灯片
	slidesXML := make(map[string][]byte)
	for i, slide := range slides {
		slideXML := generateSlideXML(slide, theme, i+1, len(slides))
		slidesXML[fmt.Sprintf("ppt/slides/slide%d.xml", i+1)] = slideXML
	}

	// 打包成 PPTX (ZIP)
	return packPPTX(title, themeXML, presXML, slidesXML, theme)
}

// generateThemeXML 生成主题 XML
func generateThemeXML(theme PPTTheme) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="%s">
  <a:themeElements>
    <a:clrScheme name="%s">
      <a:dk1><a:srgbClr val="%s"/></a:dk1>
      <a:lt1><a:srgbClr val="%s"/></a:lt1>
      <a:dk2><a:srgbClr val="%s"/></a:dk2>
      <a:lt2><a:srgbClr val="FFFFFF"/></a:lt2>
      <a:accent1><a:srgbClr val="%s"/></a:accent1>
      <a:accent2><a:srgbClr val="%s"/></a:accent2>
      <a:accent3><a:srgbClr val="60A5FA"/></a:accent3>
      <a:accent4><a:srgbClr val="34D399"/></a:accent4>
      <a:accent5><a:srgbClr val="FBBF24"/></a:accent5>
      <a:accent6><a:srgbClr val="F87171"/></a:accent6>
      <a:hlink><a:srgbClr val="2563EB"/></a:hlink>
      <a:folHlink><a:srgbClr val="1E40AF"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="%s">
      <a:majorFont><a:latin typeface="%s"/><a:ea typeface="%s"/></a:majorFont>
      <a:minorFont><a:latin typeface="%s"/><a:ea typeface="%s"/></a:minorFont>
    </a:fontScheme>
  </a:themeElements>
</a:theme>`,
		theme.Name, theme.Name,
		theme.Text, theme.Background,
		theme.Text,
		theme.Primary, theme.Accent,
		theme.Name, theme.FontHeading, theme.FontHeading,
		theme.FontBody, theme.FontBody))
}

// generatePresentationXML 生成演示文稿结构 XML
func generatePresentationXML(title string, slides []SlideContent, theme PPTTheme) []byte {
	var slideListXML strings.Builder
	baseID := 256
	for i := range slides {
		slideListXML.WriteString(fmt.Sprintf(
			`<p:sldId id="%d" r:id="rId%d"/>`,
			baseID+i, i+10))
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>%s</p:sldIdLst>
  <p:sldSz cx="9144000" cy="6858000" type="screen16x9"/>
  <p:notesSz cx="6858000" cy="9144000"/>
</p:presentation>`, slideListXML.String()))
}

// generateSlideXML 生成单张幻灯片 XML
func generateSlideXML(slide SlideContent, theme PPTTheme, num, total int) []byte {
	switch slide.Type {
	case LayoutCover:
		return generateCoverSlideXML(slide, theme, num, total)
	case LayoutSection:
		return generateSectionSlideXML(slide, theme, num, total)
	case LayoutContent:
		return generateContentSlideXML(slide, theme, num, total)
	case LayoutEnd:
		return generateEndSlideXML(slide, theme, num, total)
	default:
		return generateContentSlideXML(slide, theme, num, total)
	}
}

// generateCoverSlideXML 生成封面页
func generateCoverSlideXML(slide SlideContent, theme PPTTheme, num, total int) []byte {
	bgColor := theme.Background
	textColor := theme.Text
	if theme.Background == "0a0a1a" { // tech theme dark bg
		textColor = theme.TextLight
	}

	titleEscaped := escapeXML(slide.Title)
	subtitleEscaped := escapeXML(slide.Subtitle)

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
        <a:effectLst/>
      </p:bgPr>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <!-- 装饰形状 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="DecorBar"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="5400000"/><a:ext cx="9144000" cy="1458000"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
        </p:spPr>
      </p:sp>
      <!-- 标题 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="3" name="Title"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="1945800"/><a:ext cx="8229600" cy="972000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="zh-CN" sz="4800" b="1"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
      <!-- 副标题 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="4" name="Subtitle"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="2917800"/><a:ext cx="8229600" cy="583200"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="zh-CN" sz="2400"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`, bgColor, theme.Primary, textColor, titleEscaped, theme.Accent, subtitleEscaped))
}

// generateSectionSlideXML 生成分隔页
func generateSectionSlideXML(slide SlideContent, theme PPTTheme, num, total int) []byte {
	textColor := theme.TextLight

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
      </p:bgPr>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="2917800"/><a:ext cx="8229600" cy="972000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="zh-CN" sz="4400" b="1"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`, theme.Primary, textColor, escapeXML(slide.Title)))
}

// generateContentSlideXML 生成内容页
func generateContentSlideXML(slide SlideContent, theme PPTTheme, num, total int) []byte {
	// 构建要点列表 XML
	var bulletsXML strings.Builder
	for _, bullet := range slide.Bullets {
		if bullet == "" {
			continue
		}
		bulletEscaped := escapeXML(bullet)
		bulletsXML.WriteString(fmt.Sprintf(
			`<a:p><a:pPr lvl="0" marL="285750" indent="-285750"><a:buChar char="•"/></a:pPr>`+
			`<a:r><a:rPr lang="zh-CN" sz="2000"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr>`+
			`<a:t>%s</a:t></a:r><a:endParaRPr/></a:p>`,
			theme.Text, bulletEscaped))
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <!-- 标题栏背景 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="TitleBar"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="9144000" cy="720000"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
        </p:spPr>
      </p:sp>
      <!-- 标题 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="3" name="Title"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="228600" y="180000"/><a:ext cx="8686800" cy="450000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          <a:p>
            <a:r><a:rPr lang="zh-CN" sz="2800" b="1"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
      <!-- 内容区 -->
      <p:sp>
        <p:nvSpPr><p:cNvPr id="4" name="Content"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="228600" y="972000"/><a:ext cx="8686800" cy="4860000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          %s
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`, theme.Primary, theme.TextLight, escapeXML(slide.Title), bulletsXML.String()))
}

// generateEndSlideXML 生成结束页
func generateEndSlideXML(slide SlideContent, theme PPTTheme, num, total int) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
      </p:bgPr>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="2917800"/><a:ext cx="8229600" cy="972000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/><a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="zh-CN" sz="5400" b="1"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`, theme.Primary, theme.TextLight, escapeXML(slide.Title)))
}

// packPPTX 打包 PPTX 文件
func packPPTX(title string, themeXML, presXML []byte, slidesXML map[string][]byte, theme PPTTheme) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>`

	for i := 1; i <= len(slidesXML); i++ {
		contentTypes += fmt.Sprintf(`
  <Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i)
	}
	contentTypes += `
</Types>`

	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	// .rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`
	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(rels))

	// ppt/presentation.xml
	w, _ = zw.Create("ppt/presentation.xml")
	w.Write(presXML)

	// ppt/theme/theme1.xml
	w, _ = zw.Create("ppt/theme/theme1.xml")
	w.Write(themeXML)

	// slides
	for name, xml := range slidesXML {
		w, _ = zw.Create(name)
		w.Write(xml)
	}

	// presentation.xml.rels
	presRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>`

	for i := 1; i <= len(slidesXML); i++ {
		presRels += fmt.Sprintf(`
  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+9, i)
	}
	presRels += `
</Relationships>`

	w, _ = zw.Create("ppt/_rels/presentation.xml.rels")
	w.Write([]byte(presRels))

	zw.Close()
	return buf.Bytes(), nil
}

// escapeXML XML 特殊字符转义
func escapeXML(s string) string {
	if s == "" {
		return " "
	}
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// ParseOutlineToSlides 将 Markdown 大纲解析为 SlideContent
func ParseOutlineToSlides(outline string, themeName string) []SlideContent {
	lines := strings.Split(outline, "\n")
	var slides []SlideContent
	var currentSlide *SlideContent

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// H1 - 封面
		if strings.HasPrefix(line, "# ") {
			if currentSlide != nil {
				slides = append(slides, *currentSlide)
			}
			title := strings.TrimPrefix(line, "# ")
			currentSlide = &SlideContent{
				Type:  LayoutCover,
				Title: title,
			}
			continue
		}

		// H2 - 章节分隔页或内容页
		if strings.HasPrefix(line, "## ") {
			if currentSlide != nil && len(currentSlide.Bullets) > 0 {
				slides = append(slides, *currentSlide)
			}
			title := strings.TrimPrefix(line, "## ")
			currentSlide = &SlideContent{
				Type:  LayoutContent,
				Title: title,
			}
			continue
		}

		// 列表项 - 要点
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if currentSlide == nil {
				currentSlide = &SlideContent{Type: LayoutContent}
			}
			bullet := strings.TrimPrefix(line, "- ")
			bullet = strings.TrimPrefix(bullet, "* ")
			currentSlide.Bullets = append(currentSlide.Bullets, bullet)
		}
	}

	if currentSlide != nil {
		slides = append(slides, *currentSlide)
	}

	// 添加结束页
	slides = append(slides, SlideContent{
		Type:  LayoutEnd,
		Title: "感谢聆听",
	})

	return slides
}
