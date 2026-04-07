package notex

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const importURLMaxBody = 3 << 20 // 3 MiB

func importURLHostAllowed(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = strings.ToLower(h)
	}
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return importURLIPAllowed(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !importURLIPAllowed(ip) {
			return false
		}
	}
	return true
}

func importURLIPAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return false
		}
	}
	return true
}

func importURLPickName(rawURL *url.URL, title string) string {
	t := strings.TrimSpace(title)
	if t != "" {
		if !strings.HasSuffix(strings.ToLower(t), ".txt") {
			t += ".txt"
		}
		return t
	}
	base := path.Base(rawURL.Path)
	if base == "" || base == "." || base == "/" {
		return "url-import.txt"
	}
	if !strings.Contains(base, ".") {
		return base + ".txt"
	}
	return base
}

func (s *Server) handleDocumentsImportURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	libraryID, ok := pathInt64(r, "id")
	if !ok || libraryID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	raw := strings.TrimSpace(body.URL)
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_required"})
		return
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_url"})
		return
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_scheme_not_allowed"})
		return
	}
	host := parsed.Hostname()
	if !importURLHostAllowed(host) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_host_not_allowed"})
		return
	}

	name := importURLPickName(parsed, body.Title)

	// 先创建文档记录，状态设为pending
	var doc *Document
	if s.store != nil {
		if _, err := s.ensureDefaultLibraryForUser(r.Context(), uid); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// 创建空的pending文档
		doc, err = s.store.CreateDocumentForUser(r.Context(), uid, libraryID, name, "", 0, "application/octet-stream")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// 设置为待处理状态
		err = s.store.UpdateDocumentExtractionStatus(r.Context(), doc.ID, DocExtractionPending, "url import in progress")
		if err != nil {
			s.logger.Printf("failed to update document status: %v", err)
		}
	} else {
		s.libraryMu.Lock()
		s.documentMu.Lock()
		_ = s.ensureDefaultLibrary(uid)
		if !s.libraryBelongsToUser(uid, libraryID) {
			s.documentMu.Unlock()
			s.libraryMu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "library_not_found"})
			return
		}
		doc = &Document{
			ID:                 s.nextDocumentID,
			LibraryID:          libraryID,
			OriginalName:       name,
			Base64Data:         "",
			FileSize:           0,
			MimeType:           "application/octet-stream",
			CreatedAt:          nowRFC3339(),
			ExtractionStatus:   DocExtractionPending,
			ExtractionError:    "url import in progress",
		}
		s.nextDocumentID++
		s.documentsByID[doc.ID] = doc
		s.documentMu.Unlock()
		s.libraryMu.Unlock()
	}

	// 立刻返回响应
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"document_id": doc.ID,
		"status":      "pending",
		"message":     "URL import started, please check back later",
	})

	// 后台异步执行下载和处理
	go func(ctx context.Context, uid, libraryID int64, doc *Document, urlStr string) {
		client := &http.Client{
			Timeout: 25 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 8 {
					return fmt.Errorf("too many redirects")
				}
				if req.URL == nil {
					return nil
				}
				h := req.URL.Hostname()
				if !importURLHostAllowed(h) {
					return fmt.Errorf("redirect host not allowed")
				}
				return nil
			},
		}

		resp, err := client.Get(urlStr)
		if err != nil {
			s.logger.Printf("url import fetch failed: %v", err)
			if s.store != nil {
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, "fetch failed: "+err.Error())
			} else {
				s.documentMu.Lock()
				if d, ok := s.documentsByID[doc.ID]; ok {
					d.ExtractionStatus = DocExtractionError
					d.ExtractionError = "fetch failed: " + err.Error()
				}
				s.documentMu.Unlock()
			}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errMsg := fmt.Sprintf("http status %d", resp.StatusCode)
			s.logger.Printf("url import failed: %s", errMsg)
			if s.store != nil {
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, errMsg)
			} else {
				s.documentMu.Lock()
				if d, ok := s.documentsByID[doc.ID]; ok {
					d.ExtractionStatus = DocExtractionError
					d.ExtractionError = errMsg
				}
				s.documentMu.Unlock()
			}
			return
		}
		limited := io.LimitReader(resp.Body, importURLMaxBody+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			s.logger.Printf("url import read body failed: %v", err)
			if s.store != nil {
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, "read body failed")
			} else {
				s.documentMu.Lock()
				if d, ok := s.documentsByID[doc.ID]; ok {
					d.ExtractionStatus = DocExtractionError
					d.ExtractionError = "read body failed"
				}
				s.documentMu.Unlock()
			}
			return
		}
		if len(data) > importURLMaxBody {
			errMsg := "response too large"
			s.logger.Printf("url import failed: %s", errMsg)
			if s.store != nil {
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, errMsg)
			} else {
				s.documentMu.Lock()
				if d, ok := s.documentsByID[doc.ID]; ok {
					d.ExtractionStatus = DocExtractionError
					d.ExtractionError = errMsg
				}
				s.documentMu.Unlock()
			}
			return
		}
		if len(data) == 0 {
			errMsg := "empty response"
			s.logger.Printf("url import failed: %s", errMsg)
			if s.store != nil {
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, errMsg)
			} else {
				s.documentMu.Lock()
				if d, ok := s.documentsByID[doc.ID]; ok {
					d.ExtractionStatus = DocExtractionError
					d.ExtractionError = errMsg
				}
				s.documentMu.Unlock()
			}
			return
		}

		contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = "text/plain; charset=utf-8"
		}
		// Strip parameters for storage hint
		if i := strings.IndexByte(contentType, ';'); i >= 0 {
			contentType = strings.TrimSpace(contentType[:i])
		}

		b64 := base64.StdEncoding.EncodeToString(data)

		// 更新文档信息
		if s.store != nil {
			// 更新文档内容
			err = s.store.UpdateDocumentContent(ctx, doc.ID, b64, len(data), contentType)
			if err != nil {
				s.logger.Printf("failed to update document content: %v", err)
				_ = s.store.UpdateDocumentExtractionStatus(ctx, doc.ID, DocExtractionError, "update document failed")
				return
			}
			// 完成后续处理
			s.finalizeNewLibraryDocumentFromBytes(ctx, libraryID, doc, data, contentType)
		} else {
			s.documentMu.Lock()
			if d, ok := s.documentsByID[doc.ID]; ok {
				d.Base64Data = b64
				d.FileSize = len(data)
				d.MimeType = contentType
				d.ExtractionStatus = DocExtractionPending
				d.ExtractionError = ""
			}
			s.documentMu.Unlock()
			// 完成后续处理
			s.finalizeNewLibraryDocumentFromBytes(ctx, libraryID, doc, data, contentType)
		}

		s.logger.Printf("url import completed for document %d", doc.ID)
	}(context.Background(), uid, libraryID, doc, raw)
}
