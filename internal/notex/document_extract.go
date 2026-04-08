package notex

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat"
)

const documentExtractQueueSize = 256

// startDocumentExtractWorkers starts the background extractor and re-queues pending documents.
func (s *Server) startDocumentExtractWorkers() {
	if s.store == nil {
		return
	}
	s.docExtractOnce.Do(func() {
		s.docExtractCh = make(chan int64, documentExtractQueueSize)
		go s.documentExtractWorkerLoop()
		go s.requeuePendingDocumentExtractions()
	})
}

func (s *Server) requeuePendingDocumentExtractions() {
	ctx := context.Background()
	if _, err := s.store.ResetDocumentExtractionStaleProcessing(ctx); err != nil {
		s.logger.Printf("notex: reset stale document extraction: %v", err)
	}
	ids, err := s.store.ListDocumentIDsPendingExtraction(ctx)
	if err != nil {
		s.logger.Printf("notex: list pending document extraction: %v", err)
		return
	}
	for _, id := range ids {
		s.EnqueueDocumentExtract(id)
	}
	if len(ids) > 0 {
		s.logger.Printf("notex: re-queued %d document(s) for text extraction", len(ids))
	}
}

func (s *Server) documentExtractWorkerLoop() {
	for id := range s.docExtractCh {
		s.processDocumentExtraction(context.Background(), id)
	}
}

// EnqueueDocumentExtract schedules async text extraction (DB) or runs it in a goroutine (in-memory).
func (s *Server) EnqueueDocumentExtract(documentID int64) {
	if documentID <= 0 {
		return
	}
	if s.store != nil {
		s.startDocumentExtractWorkers()
		select {
		case s.docExtractCh <- documentID:
		default:
			s.logger.Printf("notex: document extract queue full; retry later for id %d", documentID)
		}
		return
	}
	go s.processDocumentExtraction(context.Background(), documentID)
}

func (s *Server) processDocumentExtraction(ctx context.Context, documentID int64) {
	if s.store != nil {
		s.processDocumentExtractionStore(ctx, documentID)
		return
	}
	s.processDocumentExtractionMemory(documentID)
}

func (s *Server) processDocumentExtractionStore(ctx context.Context, documentID int64) {
	doc, err := s.store.GetDocumentByID(ctx, documentID)
	if err != nil || doc == nil {
		return
	}
	if doc.ExtractionStatus == DocExtractionCompleted && strings.TrimSpace(doc.ExtractedText) != "" {
		return
	}
	if err := s.store.SetDocumentExtractionProcessing(ctx, documentID); err != nil {
		s.logger.Printf("notex: doc %d extraction processing mark: %v", documentID, err)
		return
	}
	raw, rerr := s.documentRawBytes(doc)
	if rerr != nil || len(raw) == 0 {
		_ = s.store.SetDocumentExtractionFailed(ctx, documentID, fmt.Sprintf("无法读取文档内容: %v", rerr))
		return
	}
	md, xerr := langgraphcompat.ExtractUploadedBytesToMarkdown(doc.OriginalName, doc.MimeType, raw)
	if xerr != nil {
		_ = s.store.SetDocumentExtractionFailed(ctx, documentID, xerr.Error())
		return
	}
	if strings.TrimSpace(md) == "" {
		_ = s.store.SetDocumentExtractionFailed(ctx, documentID, "解析结果为空（可能为扫描件或暂不支持的格式）")
		return
	}
	md = sanitizeTextForPostgres(md)
	if !langgraphcompat.IsPlausibleExtractedDocumentText(md) {
		_ = s.store.SetDocumentExtractionFailed(ctx, documentID,
			"解析结果疑似乱码（内置 PDF 抽取无法可靠识别此文件）。请在 config 中将 uploads.pdf_converter 设为 markitdown 并安装 markitdown CLI，或提供可复制文本的 PDF/文本版。")
		return
	}
	if err := s.store.SetDocumentExtractionDone(ctx, documentID, md); err != nil {
		s.logger.Printf("notex: doc %d save extraction: %v", documentID, err)
	}
}

func (s *Server) processDocumentExtractionMemory(documentID int64) {
	s.documentMu.Lock()
	doc, ok := s.documentsByID[documentID]
	if !ok {
		s.documentMu.Unlock()
		return
	}
	if doc.ExtractionStatus == DocExtractionCompleted && strings.TrimSpace(doc.ExtractedText) != "" {
		s.documentMu.Unlock()
		return
	}
	doc.ExtractionStatus = DocExtractionProcessing
	doc.ExtractionError = ""
	s.documentMu.Unlock()

	raw, rerr := s.documentRawBytesMemory(doc)
	if rerr != nil || len(raw) == 0 {
		s.documentMu.Lock()
		if d, ok := s.documentsByID[documentID]; ok {
			d.ExtractionStatus = DocExtractionError
			d.ExtractionError = fmt.Sprintf("无法读取文档内容: %v", rerr)
		}
		s.documentMu.Unlock()
		return
	}
	md, xerr := langgraphcompat.ExtractUploadedBytesToMarkdown(doc.OriginalName, doc.MimeType, raw)
	s.documentMu.Lock()
	defer s.documentMu.Unlock()
	d, ok := s.documentsByID[documentID]
	if !ok {
		return
	}
	if xerr != nil {
		d.ExtractionStatus = DocExtractionError
		d.ExtractionError = xerr.Error()
		return
	}
	if strings.TrimSpace(md) == "" {
		d.ExtractionStatus = DocExtractionError
		d.ExtractionError = "解析结果为空（可能为扫描件或暂不支持的格式）"
		return
	}
	md = sanitizeTextForPostgres(md)
	if !langgraphcompat.IsPlausibleExtractedDocumentText(md) {
		d.ExtractionStatus = DocExtractionError
		d.ExtractionError = "解析结果疑似乱码（内置 PDF 抽取无法可靠识别此文件）。可配置 markitdown 或提供文本版 PDF。"
		return
	}
	d.ExtractedText = md
	d.ExtractionStatus = DocExtractionCompleted
	d.ExtractionError = ""
}

func (s *Server) documentRawBytes(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil document")
	}
	if strings.TrimSpace(doc.FilePath) != "" {
		if strings.Contains(doc.FilePath, "..") {
			return nil, fmt.Errorf("invalid file path")
		}
		p := filepath.Join(s.cfg.DataRoot, filepath.FromSlash(doc.FilePath))
		return os.ReadFile(p)
	}
	if doc.Base64Data == "" {
		return nil, fmt.Errorf("empty document payload")
	}
	return base64.StdEncoding.DecodeString(doc.Base64Data)
}

func (s *Server) documentRawBytesMemory(doc *Document) ([]byte, error) {
	return s.documentRawBytes(doc)
}

// persistDocumentToDisk writes decoded upload bytes under DataRoot and updates the DB row.
func (s *Server) persistDocumentToDisk(ctx context.Context, libraryID int64, doc *Document, raw []byte, mime string) error {
	if doc == nil || len(raw) == 0 {
		return fmt.Errorf("empty payload")
	}
	safe := safeDocumentFileBase(doc.OriginalName)
	rel := path.Join("documents", fmt.Sprintf("lib_%d", libraryID), fmt.Sprintf("doc_%d_%s", doc.ID, safe))
	abs := filepath.Join(s.cfg.DataRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, raw, 0o644); err != nil {
		return err
	}
	doc.FilePath = rel
	doc.FileSize = len(raw)
	doc.MimeType = mime
	if s.store != nil {
		return s.store.UpdateDocumentFileLayout(ctx, doc.ID, rel, mime, len(raw))
	}
	return nil
}

func safeDocumentFileBase(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "/" || base == string(os.PathSeparator) {
		base = "document"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case r < 32:
			b.WriteRune('_')
		case r == '/' || r == '\\' || r == ':':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		s = "file"
	}
	if len(s) > 180 {
		ext := filepath.Ext(s)
		stem := strings.TrimSuffix(s, ext)
		if len(stem) > 160 {
			stem = stem[:160]
		}
		s = stem + ext
	}
	return s
}

// mimeTypeForOriginalName maps common extensions to MIME for extraction routing.
func mimeTypeForOriginalName(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".html", ".htm":
		return "text/html"
	case ".md", ".markdown":
		return "text/markdown"
	case ".txt", ".log":
		return "text/plain; charset=utf-8"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func (s *Server) documentChatAttachmentPayload(doc *Document) map[string]any {
	b64 := doc.Base64Data
	if strings.TrimSpace(doc.FilePath) != "" {
		if raw, err := s.documentRawBytes(doc); err == nil && len(raw) > 0 {
			b64 = base64.StdEncoding.EncodeToString(raw)
		}
	}
	return map[string]any{
		"base64_data":    b64,
		"mime_type":      doc.MimeType,
		"original_name":  doc.OriginalName,
		"file_size":      doc.FileSize,
	}
}

func (s *Server) getDocumentForStudio(ctx context.Context, documentID int64) *Document {
	if s.store != nil {
		doc, err := s.store.GetDocumentByID(ctx, documentID)
		if err != nil || doc == nil {
			return nil
		}
		return doc
	}
	s.documentMu.RLock()
	defer s.documentMu.RUnlock()
	return s.documentsByID[documentID]
}

// studioDocumentBlock builds the injected text for Studio / chat from cached extraction or legacy PDF decode.
func (s *Server) studioDocumentBlock(doc *Document) string {
	if doc == nil {
		return ""
	}
	st := doc.ExtractionStatus
	if st == "" {
		st = DocExtractionPending
	}
	switch st {
	case DocExtractionCompleted:
		text := strings.TrimSpace(doc.ExtractedText)
		if text != "" {
			if !langgraphcompat.IsPlausibleExtractedDocumentText(text) {
				return fmt.Sprintf("【知识库文档：%s】\n（正文未通过质量校验，疑似历史乱码数据，已省略注入。请删除后重新上传，或在服务器启用 markitdown PDF 解析。）\n---", doc.OriginalName)
			}
			return fmt.Sprintf("【知识库文档：%s】\n⚠️ 重要：请基于以下文档内容回答用户问题，不要使用网络搜索。\n\n%s\n---", doc.OriginalName, text)
		}
		return s.studioDocumentBlockFromBase64PDF(doc)
	case DocExtractionPending, DocExtractionProcessing:
		return fmt.Sprintf("【知识库文档：%s】\n（正文正在后台解析中，请待解析完成后再生成；当前尚无可靠全文。）\n---", doc.OriginalName)
	case DocExtractionError:
		msg := strings.TrimSpace(doc.ExtractionError)
		if msg == "" {
			msg = "未知错误"
		}
		return fmt.Sprintf("【知识库文档：%s】\n（解析失败：%s）\n---", doc.OriginalName, msg)
	default:
		return s.studioDocumentBlockFromBase64PDF(doc)
	}
}

func (s *Server) studioDocumentBlockFromBase64PDF(doc *Document) string {
	var pdfBytes []byte
	var err error
	if strings.TrimSpace(doc.Base64Data) != "" {
		pdfBytes, err = base64.StdEncoding.DecodeString(doc.Base64Data)
	} else {
		pdfBytes, err = s.documentRawBytes(doc)
	}
	if err != nil || len(pdfBytes) == 0 {
		return ""
	}
	tmpFile, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return ""
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmpFile.Write(pdfBytes); err != nil {
		_ = tmpFile.Close()
		return ""
	}
	_ = tmpFile.Close()
	text, err := langgraphcompat.ConvertPDFToMarkdown(tmpPath)
	if err != nil || strings.TrimSpace(text) == "" {
		return ""
	}
	return fmt.Sprintf("【知识库文档：%s】\n⚠️ 重要：请基于以下文档内容回答用户问题，不要使用网络搜索。\n\n%s\n---", doc.OriginalName, text)
}

// finalizeNewLibraryDocument decodes base64 from the row, writes a file under DataRoot, and queues extraction.
func (s *Server) finalizeNewLibraryDocument(ctx context.Context, libraryID int64, doc *Document) {
	if doc == nil {
		return
	}
	raw, err := base64.StdEncoding.DecodeString(doc.Base64Data)
	if err != nil || len(raw) == 0 {
		s.EnqueueDocumentExtract(doc.ID)
		return
	}
	mime := mimeTypeForOriginalName(doc.OriginalName)
	if doc.MimeType != "" && doc.MimeType != "application/octet-stream" {
		mime = doc.MimeType
	}
	s.finalizeNewLibraryDocumentFromBytes(ctx, libraryID, doc, raw, mime)
}

// finalizeNewLibraryDocumentFromBytes writes raw bytes (e.g. URL import) and queues extraction.
func (s *Server) finalizeNewLibraryDocumentFromBytes(ctx context.Context, libraryID int64, doc *Document, raw []byte, mime string) {
	if doc == nil {
		return
	}
	if len(raw) == 0 {
		s.EnqueueDocumentExtract(doc.ID)
		return
	}
	if err := s.persistDocumentToDisk(ctx, libraryID, doc, raw, mime); err != nil {
		s.logger.Printf("notex: persist document %d: %v", doc.ID, err)
	}
	s.EnqueueDocumentExtract(doc.ID)
}

// documentExtractionToParsingFields maps extraction state to legacy list API integers for the web UI.
func documentExtractionToParsingFields(st string, errMsg string) (parsingStatus, parsingProgress int, parsingError string) {
	switch st {
	case DocExtractionCompleted:
		return 2, 100, ""
	case DocExtractionProcessing:
		return 1, 50, ""
	case DocExtractionError:
		return 3, 0, errMsg
	default: // pending
		return 0, 0, ""
	}
}
