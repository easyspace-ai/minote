package notex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/milvusutil"
)

// studioPendingStaleAfter marks pending Studio rows as failed if unchanged longer than this (client refresh / disconnect).
const studioPendingStaleAfter = 20 * time.Minute

func (s *Server) libraryBelongsToUser(userID int64, libraryID int64) bool {
	for _, lib := range s.librariesByUser[userID] {
		if lib.ID == libraryID {
			return true
		}
	}
	return false
}

func (s *Server) projectBelongsToUser(userID int64, projectID int64) bool {
	for _, project := range s.projectsByUser[userID] {
		if project.ID == projectID {
			return true
		}
	}
	return false
}

func (s *Server) documentBelongsToUser(userID int64, documentID int64) bool {
	doc, exists := s.documentsByID[documentID]
	if !exists {
		return false
	}
	return s.libraryBelongsToUser(userID, doc.LibraryID)
}

func (s *Server) materialBelongsToUser(userID int64, materialID int64) bool {
	material, exists := s.materialsByID[materialID]
	if !exists {
		return false
	}
	return s.projectBelongsToUser(userID, material.ProjectID)
}

// verifyStudioPendingReplace ensures material_id refers to a pending Studio row of the expected kind when materialID > 0.
func (s *Server) verifyStudioPendingReplace(w http.ResponseWriter, ctx context.Context, uid, projectID, materialID int64, kind string) bool {
	if materialID <= 0 {
		return true
	}
	wantKind := strings.TrimSpace(kind)
	if s.store != nil {
		existing, err := s.store.GetMaterialByIDForUser(ctx, uid, materialID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return false
		}
		if existing == nil || existing.ProjectID != projectID || existing.Status != "pending" || !strings.EqualFold(strings.TrimSpace(existing.Kind), wantKind) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
			return false
		}
		return true
	}
	s.materialMu.RLock()
	defer s.materialMu.RUnlock()
	if !s.materialBelongsToUser(uid, materialID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
		return false
	}
	existing := s.materialsByID[materialID]
	if existing == nil || existing.ProjectID != projectID || existing.Status != "pending" || !strings.EqualFold(strings.TrimSpace(existing.Kind), wantKind) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
		return false
	}
	return true
}

func (s *Server) handleDocumentsUploadBrowser(w http.ResponseWriter, r *http.Request) {
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
		Files []UploadFileInput `json:"files"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if len(body.Files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "files_required"})
		return
	}
	if s.store != nil {
		if _, err := s.ensureDefaultLibraryForUser(r.Context(), uid); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		out := make([]*Document, 0, len(body.Files))
		for _, f := range body.Files {
			name := strings.TrimSpace(f.FileName)
			if name == "" {
				name = fmt.Sprintf("document-%d.txt", len(out)+1)
			}
			mime := mimeTypeForOriginalName(name)
			doc, err := s.store.CreateDocumentForUser(r.Context(), uid, libraryID, name, f.Base64Data, decodeBase64Length(f.Base64Data), mime)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			s.finalizeNewLibraryDocument(r.Context(), libraryID, doc)
			out = append(out, doc)
		}
		writeJSON(w, http.StatusCreated, out)
		return
	}

	s.libraryMu.Lock()
	s.documentMu.Lock()
	defer s.documentMu.Unlock()
	defer s.libraryMu.Unlock()
	_ = s.ensureDefaultLibrary(uid)
	if !s.libraryBelongsToUser(uid, libraryID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "library_not_found"})
		return
	}
	out := make([]*Document, 0, len(body.Files))
	for _, f := range body.Files {
		name := strings.TrimSpace(f.FileName)
		if name == "" {
			name = fmt.Sprintf("document-%d.txt", s.nextDocumentID)
		}
		mime := mimeTypeForOriginalName(name)
		doc := &Document{ID: s.nextDocumentID, LibraryID: libraryID, OriginalName: name, Base64Data: f.Base64Data, FileSize: decodeBase64Length(f.Base64Data), MimeType: mime, CreatedAt: nowRFC3339(), ExtractionStatus: DocExtractionPending}
		s.nextDocumentID++
		s.documentsByID[doc.ID] = doc
		s.finalizeNewLibraryDocument(context.Background(), libraryID, doc)
		out = append(out, doc)
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleDocumentsQuery(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	libraryID, ok := pathInt64(r, "id")
	if !ok || libraryID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	out := make([]map[string]any, 0)
	if s.store != nil {
		docs, err := s.store.ListDocumentsByLibraryForUser(r.Context(), uid, libraryID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for _, doc := range docs {
			ps, pp, perr := documentExtractionToParsingFields(doc.ExtractionStatus, doc.ExtractionError)
			es, ep := 0, 0
			if doc.ExtractionStatus == DocExtractionCompleted {
				es, ep = 2, 100
			}
			row := map[string]any{
				"id":                 doc.ID,
				"library_id":         doc.LibraryID,
				"original_name":      doc.OriginalName,
				"file_size":          doc.FileSize,
				"created_at":         doc.CreatedAt,
				"starred":            doc.Starred,
				"parsing_status":     ps,
				"parsing_progress":   pp,
				"embedding_status":   es,
				"embedding_progress": ep,
				"split_total":        0,
				"word_total":         0,
				"extraction_status":  doc.ExtractionStatus,
				"extraction_error":   doc.ExtractionError,
			}
			if perr != "" {
				row["parsing_error"] = perr
			}
			out = append(out, row)
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	s.libraryMu.RLock()
	s.documentMu.RLock()
	if !s.libraryBelongsToUser(uid, libraryID) {
		s.documentMu.RUnlock()
		s.libraryMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "library_not_found"})
		return
	}
	for _, doc := range s.documentsByID {
		if doc.LibraryID != libraryID {
			continue
		}
		ps, pp, perr := documentExtractionToParsingFields(doc.ExtractionStatus, doc.ExtractionError)
		es, ep := 0, 0
		if doc.ExtractionStatus == DocExtractionCompleted {
			es, ep = 2, 100
		}
		row := map[string]any{
			"id":                 doc.ID,
			"library_id":         doc.LibraryID,
			"original_name":      doc.OriginalName,
			"file_size":          doc.FileSize,
			"created_at":         doc.CreatedAt,
			"starred":            doc.Starred,
			"parsing_status":     ps,
			"parsing_progress":   pp,
			"embedding_status":   es,
			"embedding_progress": ep,
			"split_total":        0,
			"word_total":         0,
			"extraction_status":  doc.ExtractionStatus,
			"extraction_error":   doc.ExtractionError,
		}
		if perr != "" {
			row["parsing_error"] = perr
		}
		out = append(out, row)
	}
	s.documentMu.RUnlock()
	s.libraryMu.RUnlock()
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDocumentsVectorInspect(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		doc, err := s.store.GetDocumentByIDForUser(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if doc == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, vectorInspectResponse(r.Context(), id, doc.OriginalName))
		return
	}
	s.libraryMu.RLock()
	s.documentMu.RLock()
	if !s.documentBelongsToUser(uid, id) {
		s.documentMu.RUnlock()
		s.libraryMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	doc, exists := s.documentsByID[id]
	s.documentMu.RUnlock()
	s.libraryMu.RUnlock()
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, vectorInspectResponse(r.Context(), id, doc.OriginalName))
}

func vectorInspectResponse(ctx context.Context, documentID int64, originalName string) map[string]any {
	return map[string]any{
		"document_id": documentID,
		"chunks":      []map[string]any{{"index": 0, "text": "vector preview for " + originalName, "score": 1.0}},
		"milvus": map[string]any{
			"address_configured": milvusutil.AddressConfigured(),
			"health":             milvusutil.HealthStatus(ctx),
		},
		"vector_backend": "stub_chunks_only",
		"note":           "Chunk embedding and Milvus search are not implemented yet; use document extraction + full-text injection for chat.",
	}
}

func (s *Server) handleDocumentsChatAttachment(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		doc, err := s.store.GetDocumentByIDForUser(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if doc == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, s.documentChatAttachmentPayload(doc))
		return
	}
	s.libraryMu.RLock()
	s.documentMu.RLock()
	if !s.documentBelongsToUser(uid, id) {
		s.documentMu.RUnlock()
		s.libraryMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	doc, exists := s.documentsByID[id]
	s.documentMu.RUnlock()
	s.libraryMu.RUnlock()
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, s.documentChatAttachmentPayload(doc))
}

func (s *Server) handleDocumentsPatch(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		OriginalName *string `json:"original_name"`
		Starred      *bool   `json:"starred"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if body.OriginalName == nil && body.Starred == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_patch_fields"})
		return
	}
	if body.OriginalName != nil && strings.TrimSpace(*body.OriginalName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_original_name"})
		return
	}
	if s.store != nil {
		doc, err := s.store.PatchDocumentForUser(r.Context(), uid, id, body.OriginalName, body.Starred)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if doc == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, doc)
		return
	}
	s.libraryMu.RLock()
	s.documentMu.Lock()
	defer func() {
		s.documentMu.Unlock()
		s.libraryMu.RUnlock()
	}()
	if !s.documentBelongsToUser(uid, id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	doc := s.documentsByID[id]
	if body.OriginalName != nil {
		doc.OriginalName = strings.TrimSpace(*body.OriginalName)
	}
	if body.Starred != nil {
		doc.Starred = *body.Starred
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleDocumentsDelete(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		doc, err := s.store.GetDocumentByIDForUser(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if doc == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
			return
		}
		if strings.TrimSpace(doc.FilePath) != "" && !strings.Contains(doc.FilePath, "..") {
			_ = os.Remove(filepath.Join(s.cfg.DataRoot, filepath.FromSlash(doc.FilePath)))
		}
		if err := s.store.DeleteDocumentForUser(r.Context(), uid, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	s.libraryMu.RLock()
	s.documentMu.Lock()
	if !s.documentBelongsToUser(uid, id) {
		s.documentMu.Unlock()
		s.libraryMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		return
	}
	if d, ok := s.documentsByID[id]; ok && strings.TrimSpace(d.FilePath) != "" && !strings.Contains(d.FilePath, "..") {
		_ = os.Remove(filepath.Join(s.cfg.DataRoot, filepath.FromSlash(d.FilePath)))
	}
	delete(s.documentsByID, id)
	s.documentMu.Unlock()
	s.libraryMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	if s.store != nil {
		list, err := s.store.ListProjectsByUser(r.Context(), uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, list)
		return
	}
	s.projectMu.RLock()
	list := append([]*Project(nil), s.projectsByUser[uid]...)
	s.projectMu.RUnlock()
	sort.Slice(list, func(i, j int) bool {
		if list[i].Starred != list[j].Starred {
			return list[i].Starred
		}
		if list[i].UpdatedAt != list[j].UpdatedAt {
			return list[i].UpdatedAt > list[j].UpdatedAt
		}
		return list[i].ID > list[j].ID
	})
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleProjectsCreate(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
		return
	}
	if s.store != nil {
		// One dedicated knowledge library per project (not the user default library), so
		// conversations and documents stay scoped to this project only.
		lib, err := s.store.CreateLibrary(r.Context(), uid, name+" · 知识库", 800, 200)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		project, err := s.store.CreateProject(r.Context(), uid, lib.ID, name, strings.TrimSpace(body.Description), strings.TrimSpace(body.Category))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, project)
		return
	}
	s.libraryMu.Lock()
	s.projectMu.Lock()
	lib := &Library{ID: s.nextLibraryID, Name: name + " · 知识库", ChunkSize: 800, ChunkOverlap: 200}
	s.nextLibraryID++
	s.librariesByUser[uid] = append(s.librariesByUser[uid], lib)
	now := nowRFC3339()
	p := &Project{
		ID: s.nextProjectID, CreatedAt: now, UpdatedAt: now, Name: name,
		Description: strings.TrimSpace(body.Description), Category: strings.TrimSpace(body.Category), LibraryID: lib.ID,
		Starred: false, Archived: false, IconIndex: -1, AccentHex: "",
	}
	s.nextProjectID++
	s.projectsByUser[uid] = append(s.projectsByUser[uid], p)
	s.projectMu.Unlock()
	s.libraryMu.Unlock()
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleProjectsGet(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		project, err := s.store.GetProjectByID(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if project == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, project)
		return
	}
	s.projectMu.RLock()
	defer s.projectMu.RUnlock()
	for _, p := range s.projectsByUser[uid] {
		if p.ID == id {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
}

func (s *Server) handleProjectsDelete(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		if err := s.store.DeleteProject(r.Context(), uid, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	s.projectMu.Lock()
	list := s.projectsByUser[uid]
	filtered := list[:0]
	for _, p := range list {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	s.projectsByUser[uid] = filtered
	s.projectMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProjectsPatch(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Name        *string              `json:"name"`
		Description *string              `json:"description"`
		Category    *string              `json:"category"`
		Starred     *bool                `json:"starred"`
		Archived    *bool                `json:"archived"`
		IconIndex   *int                 `json:"icon_index"`
		AccentHex   *string              `json:"accent_hex"`
		StudioScope *StudioScopeSettings `json:"studio_scope"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if s.store != nil {
		cur, err := s.store.GetProjectByID(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if cur == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
			return
		}
		if body.Name != nil {
			n := strings.TrimSpace(*body.Name)
			if n == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
				return
			}
			cur.Name = n
		}
		if body.Description != nil {
			cur.Description = strings.TrimSpace(*body.Description)
		}
		if body.Category != nil {
			cur.Category = strings.TrimSpace(*body.Category)
		}
		if body.Starred != nil {
			cur.Starred = *body.Starred
		}
		if body.Archived != nil {
			cur.Archived = *body.Archived
		}
		if body.IconIndex != nil {
			cur.IconIndex = *body.IconIndex
		}
		if body.AccentHex != nil {
			cur.AccentHex = strings.TrimSpace(*body.AccentHex)
		}
		if body.StudioScope != nil {
			cur.StudioScope = *body.StudioScope
		}
		out, err := s.store.UpdateProjectForUser(r.Context(), uid, cur)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if out == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	list := s.projectsByUser[uid]
	for _, cur := range list {
		if cur.ID != id {
			continue
		}
		if body.Name != nil {
			n := strings.TrimSpace(*body.Name)
			if n == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
				return
			}
			cur.Name = n
		}
		if body.Description != nil {
			cur.Description = strings.TrimSpace(*body.Description)
		}
		if body.Category != nil {
			cur.Category = strings.TrimSpace(*body.Category)
		}
		if body.Starred != nil {
			cur.Starred = *body.Starred
		}
		if body.Archived != nil {
			cur.Archived = *body.Archived
		}
		if body.IconIndex != nil {
			cur.IconIndex = *body.IconIndex
		}
		if body.AccentHex != nil {
			cur.AccentHex = strings.TrimSpace(*body.AccentHex)
		}
		if body.StudioScope != nil {
			cur.StudioScope = *body.StudioScope
		}
		cur.UpdatedAt = nowRFC3339()
		writeJSON(w, http.StatusOK, cur)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
}

func (s *Server) sweepStaleStudioPendingMaterials(ctx context.Context, uid, projectID int64) {
	cutoff := time.Now().Add(-studioPendingStaleAfter)
	if s.store != nil {
		_ = s.store.MarkAbandonedStudioPendingFailed(ctx, uid, projectID, cutoff)
		return
	}
	s.projectMu.RLock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.projectMu.RUnlock()
		return
	}
	s.projectMu.RUnlock()
	s.materialMu.Lock()
	defer s.materialMu.Unlock()
	for _, m := range s.materialsByID {
		if m == nil || m.ProjectID != projectID || (m.Status != "pending" && m.Status != "processing") {
			continue
		}
		ts, err := time.Parse(time.RFC3339, m.UpdatedAt)
		if err != nil {
			continue
		}
		if ts.Before(cutoff) {
			m.Status = "failed"
			m.Subtitle = "生成已中断（页面刷新或连接断开）；请重试"
			m.UpdatedAt = nowRFC3339()
		}
	}
}

func (s *Server) handleProjectMaterialsList(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, ok := pathInt64(r, "id")
	if !ok || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	s.sweepStaleStudioPendingMaterials(r.Context(), uid, projectID)
	if s.store != nil {
		list, err := s.store.ListMaterialsByProjectForUser(r.Context(), uid, projectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, list)
		return
	}
	s.sweepStaleStudioPendingMaterials(r.Context(), uid, projectID)
	s.projectMu.RLock()
	s.materialMu.RLock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.materialMu.RUnlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	list := make([]*Material, 0)
	for _, m := range s.materialsByID {
		if m.ProjectID == projectID {
			list = append(list, m)
		}
	}
	s.materialMu.RUnlock()
	s.projectMu.RUnlock()
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleProjectMaterialsCreate(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, ok := pathInt64(r, "id")
	if !ok || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Kind     string         `json:"kind"`
		Title    string         `json:"title"`
		Status   string         `json:"status"`
		Subtitle string         `json:"subtitle"`
		Payload  map[string]any `json:"payload"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(body.Kind) == "" {
		body.Kind = "report"
	}
	if strings.TrimSpace(body.Status) == "" {
		body.Status = "ready"
	}
	if body.Payload == nil {
		body.Payload = map[string]any{}
	}
	if s.store != nil {
		material, err := s.store.CreateMaterialForUser(r.Context(), uid, projectID, strings.TrimSpace(body.Kind), strings.TrimSpace(body.Title), strings.TrimSpace(body.Status), strings.TrimSpace(body.Subtitle), body.Payload, "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, material)
		return
	}
	now := nowRFC3339()
	s.projectMu.RLock()
	s.materialMu.Lock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.materialMu.Unlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	m := &Material{ID: s.nextMaterialID, CreatedAt: now, UpdatedAt: now, ProjectID: projectID, Kind: strings.TrimSpace(body.Kind), Title: strings.TrimSpace(body.Title), Status: strings.TrimSpace(body.Status), Subtitle: strings.TrimSpace(body.Subtitle), Payload: body.Payload}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	s.materialMu.Unlock()
	s.projectMu.RUnlock()
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleProjectMaterialsGet(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	materialID, ok := pathInt64(r, "materialId")
	if !ok || materialID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
		return
	}
	if s.store != nil {
		material, err := s.store.GetMaterialByIDForUser(r.Context(), uid, materialID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if material == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, material)
		return
	}
	s.projectMu.RLock()
	s.materialMu.RLock()
	if !s.materialBelongsToUser(uid, materialID) {
		s.materialMu.RUnlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
		return
	}
	m, exists := s.materialsByID[materialID]
	s.materialMu.RUnlock()
	s.projectMu.RUnlock()
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleProjectMaterialsPatch(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, ok := pathInt64(r, "id")
	if !ok || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	materialID, ok := pathInt64(r, "materialId")
	if !ok || materialID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
		return
	}
	var body struct {
		Status   *string        `json:"status"`
		Title    *string        `json:"title"`
		Subtitle *string        `json:"subtitle"`
		Payload  map[string]any `json:"payload"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if body.Status == nil && body.Title == nil && body.Subtitle == nil && body.Payload == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty_patch"})
		return
	}
	if s.store != nil {
		cur, err := s.store.GetMaterialByIDForUser(r.Context(), uid, materialID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if cur == nil || cur.ProjectID != projectID {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		if body.Status != nil {
			cur.Status = strings.TrimSpace(*body.Status)
		}
		if body.Title != nil {
			cur.Title = strings.TrimSpace(*body.Title)
		}
		if body.Subtitle != nil {
			cur.Subtitle = strings.TrimSpace(*body.Subtitle)
		}
		if body.Payload != nil {
			cur.Payload = body.Payload
		}
		out, err := s.store.UpdateMaterialForUser(r.Context(), uid, projectID, materialID, cur.Kind, cur.Title, cur.Status, cur.Subtitle, cur.Payload, cur.FilePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if out == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	s.projectMu.RLock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	s.projectMu.RUnlock()
	s.materialMu.Lock()
	defer s.materialMu.Unlock()
	cur, exists := s.materialsByID[materialID]
	if !exists || cur == nil || cur.ProjectID != projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
		return
	}
	if body.Status != nil {
		cur.Status = strings.TrimSpace(*body.Status)
	}
	if body.Title != nil {
		cur.Title = strings.TrimSpace(*body.Title)
	}
	if body.Subtitle != nil {
		cur.Subtitle = strings.TrimSpace(*body.Subtitle)
	}
	if body.Payload != nil {
		cur.Payload = body.Payload
	}
	cur.UpdatedAt = nowRFC3339()
	writeJSON(w, http.StatusOK, cur)
}

func (s *Server) handleProjectMaterialsSlidesPPTX(w http.ResponseWriter, r *http.Request) {
	s.handleMaterialFromMarkdown(w, r, "slides", ".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
}

func (s *Server) handleProjectMaterialsStudioHTML(w http.ResponseWriter, r *http.Request) {
	s.handleMaterialFromMarkdown(w, r, "html", ".html", "text/html; charset=utf-8")
}

func (s *Server) handleProjectMaterialsStudioMindmap(w http.ResponseWriter, r *http.Request) {
	s.handleMaterialFromMarkdown(w, r, "mindmap", ".html", "text/html; charset=utf-8")
}

func (s *Server) handleProjectMaterialsStudioAudio(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, ok := pathInt64(r, "id")
	if !ok || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Title              string `json:"title"`
		Base64Data         string `json:"base64_data"`
		MimeType           string `json:"mime_type"`
		TranscriptMarkdown string `json:"transcript_markdown"`
		MaterialID         int64  `json:"material_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if !s.verifyStudioPendingReplace(w, r.Context(), uid, projectID, body.MaterialID, "audio") {
		return
	}
	s.logger.Printf(
		"[studio-audio] request project_id=%d request_id=%s title=%q base64_bytes=%d transcript_shape=%s",
		projectID,
		requestIDFromContext(r.Context()),
		strings.TrimSpace(body.Title),
		len(body.Base64Data),
		summarizeMarkdownShape(body.TranscriptMarkdown),
	)
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "播客概述"
	}
	raw, err := base64.StdEncoding.DecodeString(body.Base64Data)
	if err != nil || len(raw) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_base64_audio"})
		return
	}
	mime := strings.TrimSpace(body.MimeType)
	if mime == "" {
		mime = "audio/mpeg"
	}
	ext := ".mp3"
	if strings.Contains(strings.ToLower(mime), "wav") {
		ext = ".wav"
	} else if strings.Contains(strings.ToLower(mime), "ogg") {
		ext = ".ogg"
	} else if strings.Contains(strings.ToLower(mime), "aac") {
		ext = ".aac"
	}
	filename := ensureExtension(strings.ReplaceAll(strings.ToLower(title), " ", "-"), ext)
	filePath, err := s.writeMaterialFile(projectID, filename, raw)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.logger.Printf(
		"[studio-audio] file persisted project_id=%d request_id=%s path=%q bytes=%d mime=%s",
		projectID,
		requestIDFromContext(r.Context()),
		filePath,
		len(raw),
		mime,
	)
	subtitle := "AI 播客 · TTS"
	transcript := strings.TrimSpace(body.TranscriptMarkdown)
	payload := map[string]any{
		"file_name": filename,
		"mime_type": mime,
		"markdown":  transcript,
	}
	if s.store != nil {
		if body.MaterialID > 0 {
			material, err := s.store.UpdateMaterialForUser(r.Context(), uid, projectID, body.MaterialID, "audio", title, "ready", subtitle, payload, filePath)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if material == nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
				return
			}
			s.logger.Printf(
				"[studio-audio] material finalized project_id=%d request_id=%s material_id=%d payload=%v",
				projectID,
				requestIDFromContext(r.Context()),
				material.ID,
				payload,
			)
			writeJSON(w, http.StatusOK, material)
			return
		}
		material, err := s.store.CreateMaterialForUser(r.Context(), uid, projectID, "audio", title, "ready", subtitle, payload, filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.logger.Printf(
			"[studio-audio] material created project_id=%d request_id=%s material_id=%d payload=%v",
			projectID,
			requestIDFromContext(r.Context()),
			material.ID,
			payload,
		)
		writeJSON(w, http.StatusCreated, material)
		return
	}
	if body.MaterialID > 0 {
		now := nowRFC3339()
		s.projectMu.RLock()
		s.materialMu.Lock()
		if !s.projectBelongsToUser(uid, projectID) {
			s.materialMu.Unlock()
			s.projectMu.RUnlock()
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
			return
		}
		m := s.materialsByID[body.MaterialID]
		if m == nil {
			s.materialMu.Unlock()
			s.projectMu.RUnlock()
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		m.Kind = "audio"
		m.Title = title
		m.Status = "ready"
		m.Subtitle = subtitle
		m.Payload = payload
		m.FilePath = filePath
		m.UpdatedAt = now
		s.materialMu.Unlock()
		s.projectMu.RUnlock()
		s.logger.Printf(
			"[studio-audio] material finalized project_id=%d request_id=%s material_id=%d payload=%v",
			projectID,
			requestIDFromContext(r.Context()),
			m.ID,
			payload,
		)
		writeJSON(w, http.StatusOK, m)
		return
	}
	now := nowRFC3339()
	s.projectMu.RLock()
	s.materialMu.Lock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.materialMu.Unlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	m := &Material{ID: s.nextMaterialID, CreatedAt: now, UpdatedAt: now, ProjectID: projectID, Kind: "audio", Title: title, Status: "ready", Subtitle: subtitle, Payload: payload, FilePath: filePath}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	s.materialMu.Unlock()
	s.projectMu.RUnlock()
	s.logger.Printf(
		"[studio-audio] material created project_id=%d request_id=%s material_id=%d payload=%v",
		projectID,
		requestIDFromContext(r.Context()),
		m.ID,
		payload,
	)
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleMaterialFromMarkdown(w http.ResponseWriter, r *http.Request, kind string, ext string, mime string) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, ok := pathInt64(r, "id")
	if !ok || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Title          string `json:"title"`
		Markdown       string `json:"markdown"`
		Language       string `json:"language"`
		ConversationID int64  `json:"conversation_id"`
		MaterialID     int64  `json:"material_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if !s.verifyStudioPendingReplace(w, r.Context(), uid, projectID, body.MaterialID, kind) {
		return
	}
	s.logger.Printf(
		"[studio-material] request kind=%s project_id=%d request_id=%s conversation_id=%d title=%q markdown_bytes=%d markdown_shape=%s",
		kind,
		projectID,
		requestIDFromContext(r.Context()),
		body.ConversationID,
		strings.TrimSpace(body.Title),
		len(body.Markdown),
		summarizeMarkdownShape(body.Markdown),
	)
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = strings.Title(kind)
	}
	filename := ensureExtension(strings.ReplaceAll(strings.ToLower(title), " ", "-"), ext)
	var fileBytes []byte
	var slidePPTXSource string
	var slideSkillVPath string
	switch {
	case strings.EqualFold(ext, ".pptx"):
		// Strict: only accept a skill-produced .pptx from the LangGraph thread (no Markdown→OOXML fallback).
		if body.ConversationID <= 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "pptx_skill_artifact_missing: conversation_id is required to resolve thread .pptx artifacts",
			})
			return
		}
		preferred, vpath, prefErr := s.preferredStudioPPTXFromConversation(r.Context(), uid, body.ConversationID)
		if prefErr != nil {
			s.logger.Printf("[studio-material] slides skill lookup error: project_id=%d conversation_id=%d err=%v",
				projectID, body.ConversationID, prefErr)
		}
		if len(preferred) == 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "pptx_skill_artifact_missing: no valid .pptx on this conversation's LangGraph thread; the agent must emit a real PPTX (e.g. under user-data/outputs/) before calling slides-pptx",
			})
			return
		}
		fileBytes = preferred
		slidePPTXSource = "thread_skill"
		slideSkillVPath = vpath
		s.logger.Printf(
			"[studio-material] slides source selected=thread_skill project_id=%d request_id=%s conversation_id=%d artifact_path=%q bytes=%d",
			projectID,
			requestIDFromContext(r.Context()),
			body.ConversationID,
			vpath,
			len(fileBytes),
		)
	case strings.HasSuffix(ext, ".html"):
		md := strings.TrimSpace(body.Markdown)
		var content string
		if kind == "html" {
			// AI is expected to output HTML (body fragment or full document); embed it directly.
			lower := strings.ToLower(md)
			if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
				content = md
			} else {
				content = "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>" + title + "</title></head>\n<body>\n" + md + "\n</body></html>"
			}
		} else if kind == "mindmap" {
			// Markmap autoloader renders nested-list Markdown as an interactive mind map.
			content = "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>" + title + "</title>" +
				"<style>body{margin:0;overflow:hidden}svg.markmap{width:100vw;height:100vh}</style>" +
				"<script src=\"https://cdn.jsdelivr.net/npm/markmap-autoloader@latest\"></script>" +
				"</head>\n<body>\n<div class=\"markmap\">\n<script type=\"text/template\">\n" +
				md + "\n</script>\n</div>\n</body></html>"
		} else {
			content = "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>" + title + "</title></head>\n<body><pre style=\"white-space:pre-wrap;font-family:monospace\">\n" + md + "\n</pre></body></html>"
		}
		fileBytes = []byte(content)
	default:
		fileBytes = []byte(body.Markdown)
	}
	filePath, err := s.writeMaterialFile(projectID, filename, fileBytes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.logger.Printf(
		"[studio-material] file persisted kind=%s project_id=%d request_id=%s path=%q bytes=%d",
		kind,
		projectID,
		requestIDFromContext(r.Context()),
		filePath,
		len(fileBytes),
	)
	payload := map[string]any{"file_name": filename, "mime_type": mime}
	// Preserve source Markdown for Studio preview (slides outline, HTML source, mindmap list, etc.).
	if md := strings.TrimSpace(body.Markdown); md != "" {
		payload["markdown"] = md
	}
	if kind == "slides" {
		meta := map[string]any{"available": true, "file_name": filename}
		if slidePPTXSource != "" {
			meta["source"] = slidePPTXSource
		}
		if slideSkillVPath != "" {
			meta["skill_artifact_path"] = slideSkillVPath
		}
		payload["pptx"] = meta
	}
	if s.store != nil {
		if body.MaterialID > 0 {
			material, err := s.store.UpdateMaterialForUser(r.Context(), uid, projectID, body.MaterialID, kind, title, "ready", "", payload, filePath)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if material == nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
				return
			}
			s.logger.Printf(
				"[studio-material] material finalized kind=%s project_id=%d request_id=%s material_id=%d payload=%v",
				kind,
				projectID,
				requestIDFromContext(r.Context()),
				material.ID,
				payload,
			)
			writeJSON(w, http.StatusOK, material)
			return
		}
		material, err := s.store.CreateMaterialForUser(r.Context(), uid, projectID, kind, title, "ready", "", payload, filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.logger.Printf(
			"[studio-material] material created kind=%s project_id=%d request_id=%s material_id=%d payload=%v",
			kind,
			projectID,
			requestIDFromContext(r.Context()),
			material.ID,
			payload,
		)
		writeJSON(w, http.StatusCreated, material)
		return
	}
	if body.MaterialID > 0 {
		now := nowRFC3339()
		s.projectMu.RLock()
		s.materialMu.Lock()
		if !s.projectBelongsToUser(uid, projectID) {
			s.materialMu.Unlock()
			s.projectMu.RUnlock()
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
			return
		}
		m := s.materialsByID[body.MaterialID]
		if m == nil {
			s.materialMu.Unlock()
			s.projectMu.RUnlock()
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		m.Kind = kind
		m.Title = title
		m.Status = "ready"
		m.Subtitle = ""
		m.Payload = payload
		m.FilePath = filePath
		m.UpdatedAt = now
		s.materialMu.Unlock()
		s.projectMu.RUnlock()
		s.logger.Printf(
			"[studio-material] material finalized kind=%s project_id=%d request_id=%s material_id=%d payload=%v",
			kind,
			projectID,
			requestIDFromContext(r.Context()),
			m.ID,
			payload,
		)
		writeJSON(w, http.StatusOK, m)
		return
	}
	now := nowRFC3339()
	s.projectMu.RLock()
	s.materialMu.Lock()
	if !s.projectBelongsToUser(uid, projectID) {
		s.materialMu.Unlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	m := &Material{ID: s.nextMaterialID, CreatedAt: now, UpdatedAt: now, ProjectID: projectID, Kind: kind, Title: title, Status: "ready", Payload: payload, FilePath: filePath}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	s.materialMu.Unlock()
	s.projectMu.RUnlock()
	s.logger.Printf(
		"[studio-material] material created kind=%s project_id=%d request_id=%s material_id=%d payload=%v",
		kind,
		projectID,
		requestIDFromContext(r.Context()),
		m.ID,
		payload,
	)
	writeJSON(w, http.StatusCreated, m)
}

func summarizeMarkdownShape(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var nonEmpty, h1, h2, h3p, dashBullets, numbered int
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		nonEmpty++
		switch {
		case strings.HasPrefix(t, "# "):
			h1++
		case strings.HasPrefix(t, "## "):
			h2++
		case strings.HasPrefix(t, "### ") || strings.HasPrefix(t, "#### ") || strings.HasPrefix(t, "##### ") || strings.HasPrefix(t, "###### "):
			h3p++
		}
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			dashBullets++
		}
		if n := strings.Index(t, ". "); n > 0 {
			isDigits := true
			for i := 0; i < n; i++ {
				if t[i] < '0' || t[i] > '9' {
					isDigits = false
					break
				}
			}
			if isDigits {
				numbered++
			}
		}
	}
	preview := strings.TrimSpace(md)
	if len(preview) > 220 {
		preview = preview[:220] + "..."
	}
	preview = strings.ReplaceAll(preview, "\n", "\\n")
	return fmt.Sprintf("non_empty=%d h1=%d h2=%d h3+=%d bullets=%d numbered=%d preview=%q", nonEmpty, h1, h2, h3p, dashBullets, numbered, preview)
}

func (s *Server) handleProjectMaterialPPTXDownload(w http.ResponseWriter, r *http.Request) {
	s.downloadMaterialFile(w, r, "slides", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
}

func (s *Server) handleProjectMaterialStudioFileDownload(w http.ResponseWriter, r *http.Request) {
	s.downloadMaterialFile(w, r, "", "")
}

// studioExportContentType returns MIME for GET .../studio-file (HTML exports, markmap, or stored audio).
func studioExportContentType(m *Material, fallback string) string {
	if m != nil && m.Payload != nil {
		if v, ok := m.Payload["mime_type"].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	if m != nil && m.Kind == "audio" {
		return "audio/mpeg"
	}
	if fallback != "" {
		return fallback
	}
	return "text/html; charset=utf-8"
}

func (s *Server) downloadMaterialFile(w http.ResponseWriter, r *http.Request, kind string, contentType string) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	materialID, ok := pathInt64(r, "materialId")
	if !ok || materialID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_id"})
		return
	}
	if s.store != nil {
		material, err := s.store.GetMaterialByIDForUser(r.Context(), uid, materialID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if material == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
			return
		}
		if kind != "" && material.Kind != kind {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_kind"})
			return
		}
		if material.FilePath == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_file_not_found"})
			return
		}
		b, err := os.ReadFile(material.FilePath)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_file_not_found"})
			return
		}
		ct := contentType
		if kind == "" {
			ct = studioExportContentType(material, contentType)
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		disp := "attachment"
		if material.Kind == "audio" {
			disp = "inline"
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename=%q`, disp, filepath.Base(material.FilePath)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
		return
	}
	s.projectMu.RLock()
	s.materialMu.RLock()
	if !s.materialBelongsToUser(uid, materialID) {
		s.materialMu.RUnlock()
		s.projectMu.RUnlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
		return
	}
	m, exists := s.materialsByID[materialID]
	s.materialMu.RUnlock()
	s.projectMu.RUnlock()
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_not_found"})
		return
	}
	if kind != "" && m.Kind != kind {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_material_kind"})
		return
	}
	if m.FilePath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_file_not_found"})
		return
	}
	b, err := os.ReadFile(m.FilePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "material_file_not_found"})
		return
	}
	ct := contentType
	if kind == "" {
		ct = studioExportContentType(m, contentType)
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	disp := "attachment"
	if m.Kind == "audio" {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename=%q`, disp, filepath.Base(m.FilePath)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
