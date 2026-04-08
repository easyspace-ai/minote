package notex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func decodeStudioScope(raw []byte) (StudioScopeSettings, error) {
	var s StudioScopeSettings
	if len(raw) == 0 || string(raw) == "null" {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decode studio_scope: %w", err)
	}
	return s, nil
}

func (s *Store) ListProjectsByUser(ctx context.Context, userID int64) ([]*Project, error) {
	rows, err := s.db.Query(ctx, `
		select id, created_at, updated_at, name, description, category, library_id,
		       starred, archived, icon_index, accent_hex, studio_scope
		from notex_projects
		where user_id = $1
		order by starred desc, updated_at desc, id desc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []*Project
	for rows.Next() {
		var (
			project              Project
			createdAt, updatedAt time.Time
			scopeRaw             []byte
		)
		if err := rows.Scan(&project.ID, &createdAt, &updatedAt, &project.Name, &project.Description, &project.Category, &project.LibraryID,
			&project.Starred, &project.Archived, &project.IconIndex, &project.AccentHex, &scopeRaw); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		sc, err := decodeStudioScope(scopeRaw)
		if err != nil {
			return nil, err
		}
		project.StudioScope = sc
		project.CreatedAt = scanTimestampRFC3339(createdAt)
		project.UpdatedAt = scanTimestampRFC3339(updatedAt)
		out = append(out, &project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return out, nil
}

func (s *Store) CreateProject(ctx context.Context, userID int64, libraryID int64, name string, description string, category string) (*Project, error) {
	var (
		project              Project
		createdAt, updatedAt time.Time
	)
	var scopeRaw []byte
	projectUUID := newProjectUUID()
	err := s.db.QueryRow(ctx, `
		insert into notex_projects (id, user_id, library_id, name, description, category, created_at, updated_at)
		select $1, $2, l.id, $4, $5, $6, now(), now()
		from notex_libraries l
		where l.user_id = $2 and l.id = $3
		returning id, created_at, updated_at, name, description, category, library_id,
		          starred, archived, icon_index, accent_hex, studio_scope
	`, projectUUID, userID, libraryID, name, description, category).Scan(
		&project.ID,
		&createdAt,
		&updatedAt,
		&project.Name,
		&project.Description,
		&project.Category,
		&project.LibraryID,
		&project.Starred,
		&project.Archived,
		&project.IconIndex,
		&project.AccentHex,
		&scopeRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	sc, err := decodeStudioScope(scopeRaw)
	if err != nil {
		return nil, err
	}
	project.StudioScope = sc
	project.CreatedAt = scanTimestampRFC3339(createdAt)
	project.UpdatedAt = scanTimestampRFC3339(updatedAt)
	return &project, nil
}

func (s *Store) GetProjectByID(ctx context.Context, userID int64, projectID string) (*Project, error) {
	var (
		project              Project
		createdAt, updatedAt time.Time
		scopeRaw             []byte
	)
	err := s.db.QueryRow(ctx, `
		select id, created_at, updated_at, name, description, category, library_id,
		       starred, archived, icon_index, accent_hex, studio_scope
		from notex_projects
		where user_id = $1 and id = $2
	`, userID, projectID).Scan(&project.ID, &createdAt, &updatedAt, &project.Name, &project.Description, &project.Category, &project.LibraryID,
		&project.Starred, &project.Archived, &project.IconIndex, &project.AccentHex, &scopeRaw)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	sc, err := decodeStudioScope(scopeRaw)
	if err != nil {
		return nil, err
	}
	project.StudioScope = sc
	project.CreatedAt = scanTimestampRFC3339(createdAt)
	project.UpdatedAt = scanTimestampRFC3339(updatedAt)
	return &project, nil
}

func (s *Store) DeleteProject(ctx context.Context, userID int64, projectID string) error {
	if _, err := s.db.Exec(ctx, `delete from notex_projects where user_id = $1 and id = $2`, userID, projectID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// UpdateProjectForUser replaces mutable fields (caller loads and merges first).
func (s *Store) UpdateProjectForUser(ctx context.Context, userID int64, p *Project) (*Project, error) {
	scopeRaw, err := json.Marshal(p.StudioScope)
	if err != nil {
		return nil, fmt.Errorf("marshal studio_scope: %w", err)
	}
	var (
		out                  Project
		createdAt, updatedAt time.Time
		outScopeRaw          []byte
	)
	err = s.db.QueryRow(ctx, `
		update notex_projects set
			name = $3,
			description = $4,
			category = $5,
			starred = $6,
			archived = $7,
			icon_index = $8,
			accent_hex = $9,
			studio_scope = $10::jsonb,
			updated_at = now()
		where user_id = $1 and id = $2
		returning id, created_at, updated_at, name, description, category, library_id,
		          starred, archived, icon_index, accent_hex, studio_scope
	`, userID, p.ID, p.Name, p.Description, p.Category, p.Starred, p.Archived, p.IconIndex, p.AccentHex, scopeRaw).Scan(
		&out.ID,
		&createdAt,
		&updatedAt,
		&out.Name,
		&out.Description,
		&out.Category,
		&out.LibraryID,
		&out.Starred,
		&out.Archived,
		&out.IconIndex,
		&out.AccentHex,
		&outScopeRaw,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("update project: %w", err)
	}
	sc, err := decodeStudioScope(outScopeRaw)
	if err != nil {
		return nil, err
	}
	out.StudioScope = sc
	out.CreatedAt = scanTimestampRFC3339(createdAt)
	out.UpdatedAt = scanTimestampRFC3339(updatedAt)
	return &out, nil
}

func (s *Store) CreateDocument(ctx context.Context, libraryID int64, originalName string, base64Data string, fileSize int, mimeType string) (*Document, error) {
	return s.CreateDocumentForUser(ctx, 0, libraryID, originalName, base64Data, fileSize, mimeType)
}

func (s *Store) CreateDocumentForUser(ctx context.Context, userID int64, libraryID int64, originalName string, base64Data string, fileSize int, mimeType string) (*Document, error) {
	var (
		document  Document
		createdAt time.Time
	)
	query := `
		insert into notex_documents (library_id, original_name, base64_data, file_size, mime_type, created_at, updated_at)
		values ($1, $2, $3, $4, $5, now(), now())
		returning id, library_id, original_name, base64_data, file_size, mime_type, created_at, starred,
			file_path, extracted_text, extraction_status, extraction_error
	`
	args := []any{libraryID, originalName, base64Data, fileSize, mimeType}
	if userID > 0 {
		query = `
			insert into notex_documents (library_id, original_name, base64_data, file_size, mime_type, created_at, updated_at)
			select l.id, $3, $4, $5, $6, now(), now()
			from notex_libraries l
			where l.user_id = $1 and l.id = $2
			returning id, library_id, original_name, base64_data, file_size, mime_type, created_at, starred,
				file_path, extracted_text, extraction_status, extraction_error
		`
		args = []any{userID, libraryID, originalName, base64Data, fileSize, mimeType}
	}
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&document.ID,
		&document.LibraryID,
		&document.OriginalName,
		&document.Base64Data,
		&document.FileSize,
		&document.MimeType,
		&createdAt,
		&document.Starred,
		&document.FilePath,
		&document.ExtractedText,
		&document.ExtractionStatus,
		&document.ExtractionError,
	)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	document.CreatedAt = scanTimestampRFC3339(createdAt)
	if document.ExtractionStatus == "" {
		document.ExtractionStatus = DocExtractionPending
	}
	return &document, nil
}

func (s *Store) ListDocumentsByLibrary(ctx context.Context, libraryID int64) ([]*Document, error) {
	return s.ListDocumentsByLibraryForUser(ctx, 0, libraryID)
}

func (s *Store) ListDocumentsByLibraryForUser(ctx context.Context, userID int64, libraryID int64) ([]*Document, error) {
	query := `
		select d.id, d.library_id, d.original_name, d.base64_data, d.file_size, d.mime_type, d.created_at, d.starred,
			d.file_path, d.extracted_text, d.extraction_status, d.extraction_error
		from notex_documents d
		where d.library_id = $1
		order by d.starred desc, d.id desc
	`
	args := []any{libraryID}
	if userID > 0 {
		query = `
			select d.id, d.library_id, d.original_name, d.base64_data, d.file_size, d.mime_type, d.created_at, d.starred,
				d.file_path, d.extracted_text, d.extraction_status, d.extraction_error
			from notex_documents d
			join notex_libraries l on l.id = d.library_id
			where l.user_id = $1 and d.library_id = $2
			order by d.starred desc, d.id desc
		`
		args = []any{userID, libraryID}
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []*Document
	for rows.Next() {
		var (
			document  Document
			createdAt time.Time
		)
		if err := rows.Scan(&document.ID, &document.LibraryID, &document.OriginalName, &document.Base64Data, &document.FileSize, &document.MimeType, &createdAt, &document.Starred,
			&document.FilePath, &document.ExtractedText, &document.ExtractionStatus, &document.ExtractionError); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		document.CreatedAt = scanTimestampRFC3339(createdAt)
		if document.ExtractionStatus == "" {
			document.ExtractionStatus = DocExtractionPending
		}
		out = append(out, &document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return out, nil
}

func (s *Store) GetDocumentByID(ctx context.Context, documentID int64) (*Document, error) {
	return s.GetDocumentByIDForUser(ctx, 0, documentID)
}

func (s *Store) GetDocumentByIDForUser(ctx context.Context, userID int64, documentID int64) (*Document, error) {
	var (
		document  Document
		createdAt time.Time
	)
	query := `
		select d.id, d.library_id, d.original_name, d.base64_data, d.file_size, d.mime_type, d.created_at, d.starred,
			d.file_path, d.extracted_text, d.extraction_status, d.extraction_error
		from notex_documents d
		where d.id = $1
	`
	args := []any{documentID}
	if userID > 0 {
		query = `
			select d.id, d.library_id, d.original_name, d.base64_data, d.file_size, d.mime_type, d.created_at, d.starred,
				d.file_path, d.extracted_text, d.extraction_status, d.extraction_error
			from notex_documents d
			join notex_libraries l on l.id = d.library_id
			where l.user_id = $1 and d.id = $2
		`
		args = []any{userID, documentID}
	}
	err := s.db.QueryRow(ctx, query, args...).Scan(&document.ID, &document.LibraryID, &document.OriginalName, &document.Base64Data, &document.FileSize, &document.MimeType, &createdAt, &document.Starred,
		&document.FilePath, &document.ExtractedText, &document.ExtractionStatus, &document.ExtractionError)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get document: %w", err)
	}
	document.CreatedAt = scanTimestampRFC3339(createdAt)
	if document.ExtractionStatus == "" {
		document.ExtractionStatus = DocExtractionPending
	}
	return &document, nil
}

func (s *Store) DeleteDocument(ctx context.Context, documentID int64) error {
	return s.DeleteDocumentForUser(ctx, 0, documentID)
}

func (s *Store) DeleteDocumentForUser(ctx context.Context, userID int64, documentID int64) error {
	query := `delete from notex_documents where id = $1`
	args := []any{documentID}
	if userID > 0 {
		query = `
			delete from notex_documents d
			using notex_libraries l
			where d.library_id = l.id and l.user_id = $1 and d.id = $2
		`
		args = []any{userID, documentID}
	}
	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

// PatchDocumentForUser updates display name and/or starred; nil pointers leave fields unchanged.
func (s *Store) PatchDocumentForUser(ctx context.Context, userID int64, documentID int64, originalName *string, starred *bool) (*Document, error) {
	if originalName == nil && starred == nil {
		return nil, fmt.Errorf("no fields to patch")
	}
	var nameArg any
	if originalName != nil {
		t := strings.TrimSpace(*originalName)
		if t == "" {
			return nil, fmt.Errorf("original_name empty")
		}
		nameArg = t
	}
	var starArg any
	if starred != nil {
		starArg = *starred
	}
	var (
		document  Document
		createdAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		update notex_documents d set
			original_name = coalesce($3::text, original_name),
			starred = coalesce($4::boolean, starred),
			updated_at = now()
		from notex_libraries l
		where d.library_id = l.id and l.user_id = $1 and d.id = $2
		returning d.id, d.library_id, d.original_name, d.base64_data, d.file_size, d.mime_type, d.created_at, d.starred,
			d.file_path, d.extracted_text, d.extraction_status, d.extraction_error
	`, userID, documentID, nameArg, starArg).Scan(
		&document.ID,
		&document.LibraryID,
		&document.OriginalName,
		&document.Base64Data,
		&document.FileSize,
		&document.MimeType,
		&createdAt,
		&document.Starred,
		&document.FilePath,
		&document.ExtractedText,
		&document.ExtractionStatus,
		&document.ExtractionError,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("patch document: %w", err)
	}
	document.CreatedAt = scanTimestampRFC3339(createdAt)
	if document.ExtractionStatus == "" {
		document.ExtractionStatus = DocExtractionPending
	}
	return &document, nil
}

// UpdateDocumentFileLayout sets on-disk path and refreshed mime/size after persisting raw bytes under DataRoot.
func (s *Store) UpdateDocumentFileLayout(ctx context.Context, documentID int64, relativePath, mimeType string, fileSize int) error {
	if _, err := s.db.Exec(ctx, `
		update notex_documents set file_path = $2, mime_type = $3, file_size = $4, updated_at = now() where id = $1
	`, documentID, relativePath, mimeType, fileSize); err != nil {
		return fmt.Errorf("update document file layout: %w", err)
	}
	return nil
}

func (s *Store) SetDocumentExtractionProcessing(ctx context.Context, documentID int64) error {
	if _, err := s.db.Exec(ctx, `
		update notex_documents set extraction_status = 'processing', extraction_error = '', updated_at = now() where id = $1
	`, documentID); err != nil {
		return fmt.Errorf("set document extraction processing: %w", err)
	}
	return nil
}

func (s *Store) SetDocumentExtractionDone(ctx context.Context, documentID int64, markdown string) error {
	markdown = sanitizeTextForPostgres(markdown)
	if _, err := s.db.Exec(ctx, `
		update notex_documents set extraction_status = 'completed', extraction_error = '', extracted_text = $2, updated_at = now() where id = $1
	`, documentID, markdown); err != nil {
		return fmt.Errorf("set document extraction done: %w", err)
	}
	return nil
}

func (s *Store) SetDocumentExtractionFailed(ctx context.Context, documentID int64, errMsg string) error {
	errMsg = sanitizeTextForPostgres(errMsg)
	if _, err := s.db.Exec(ctx, `
		update notex_documents set extraction_status = 'error', extraction_error = $2, updated_at = now() where id = $1
	`, documentID, errMsg); err != nil {
		return fmt.Errorf("set document extraction failed: %w", err)
	}
	return nil
}

// UpdateDocumentExtractionStatus sets extraction_status and extraction_error (e.g. URL import progress).
func (s *Store) UpdateDocumentExtractionStatus(ctx context.Context, documentID int64, status string, errMsg string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		status = DocExtractionPending
	}
	errMsg = sanitizeTextForPostgres(errMsg)
	if _, err := s.db.Exec(ctx, `
		update notex_documents set extraction_status = $2, extraction_error = $3, updated_at = now() where id = $1
	`, documentID, status, errMsg); err != nil {
		return fmt.Errorf("update document extraction status: %w", err)
	}
	return nil
}

// UpdateDocumentContent replaces inline base64 payload and related fields after URL import or similar.
func (s *Store) UpdateDocumentContent(ctx context.Context, documentID int64, base64Data string, fileSize int, mimeType string) error {
	if _, err := s.db.Exec(ctx, `
		update notex_documents set base64_data = $2, file_size = $3, mime_type = $4, updated_at = now() where id = $1
	`, documentID, base64Data, fileSize, strings.TrimSpace(mimeType)); err != nil {
		return fmt.Errorf("update document content: %w", err)
	}
	return nil
}

// ResetDocumentExtractionStaleProcessing marks interrupted jobs as pending (e.g. after server restart).
func (s *Store) ResetDocumentExtractionStaleProcessing(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `update notex_documents set extraction_status = $1 where extraction_status = $2`, DocExtractionPending, DocExtractionProcessing)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListDocumentIDsPendingExtraction returns documents waiting for background text extraction.
func (s *Store) ListDocumentIDsPendingExtraction(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Query(ctx, `select id from notex_documents where extraction_status = $1 order by id asc`, DocExtractionPending)
	if err != nil {
		return nil, fmt.Errorf("list pending extraction: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) ListMaterialsByProject(ctx context.Context, projectID string) ([]*Material, error) {
	return s.ListMaterialsByProjectForUser(ctx, 0, projectID)
}

func (s *Store) ListMaterialsByProjectForUser(ctx context.Context, userID int64, projectID string) ([]*Material, error) {
	query := `
		select m.id, m.created_at, m.updated_at, m.project_id, m.kind, m.title, m.status, m.subtitle, m.payload, m.file_path
		from notex_materials m
		where m.project_id = $1
		order by m.id asc
	`
	args := []any{projectID}
	if userID > 0 {
		query = `
			select m.id, m.created_at, m.updated_at, m.project_id, m.kind, m.title, m.status, m.subtitle, m.payload, m.file_path
			from notex_materials m
			join notex_projects p on p.id = m.project_id
			where p.user_id = $1 and m.project_id = $2
			order by m.id asc
		`
		args = []any{userID, projectID}
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	defer rows.Close()

	var out []*Material
	for rows.Next() {
		var (
			material             Material
			createdAt, updatedAt time.Time
			payloadRaw           []byte
		)
		if err := rows.Scan(&material.ID, &createdAt, &updatedAt, &material.ProjectID, &material.Kind, &material.Title, &material.Status, &material.Subtitle, &payloadRaw, &material.FilePath); err != nil {
			return nil, fmt.Errorf("scan material: %w", err)
		}
		material.CreatedAt = scanTimestampRFC3339(createdAt)
		material.UpdatedAt = scanTimestampRFC3339(updatedAt)
		if len(payloadRaw) > 0 {
			if err := json.Unmarshal(payloadRaw, &material.Payload); err != nil {
				return nil, fmt.Errorf("decode material payload: %w", err)
			}
		}
		if material.Payload == nil {
			material.Payload = map[string]any{}
		}
		out = append(out, &material)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate materials: %w", err)
	}
	return out, nil
}

func (s *Store) CreateMaterial(ctx context.Context, projectID string, kind string, title string, status string, subtitle string, payload map[string]any, filePath string) (*Material, error) {
	return s.CreateMaterialForUser(ctx, 0, projectID, kind, title, status, subtitle, payload, filePath)
}

func (s *Store) CreateMaterialForUser(ctx context.Context, userID int64, projectID string, kind string, title string, status string, subtitle string, payload map[string]any, filePath string) (*Material, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal material payload: %w", err)
	}
	var (
		material             Material
		createdAt, updatedAt time.Time
		storedPayload        []byte
	)
	query := `
		insert into notex_materials (project_id, kind, title, status, subtitle, payload, file_path, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, now(), now())
		returning id, created_at, updated_at, project_id, kind, title, status, subtitle, payload, file_path
	`
	args := []any{projectID, kind, title, status, subtitle, payloadRaw, filePath}
	if userID > 0 {
		query = `
			insert into notex_materials (project_id, kind, title, status, subtitle, payload, file_path, created_at, updated_at)
			select p.id, $3, $4, $5, $6, $7, $8, now(), now()
			from notex_projects p
			where p.user_id = $1 and p.id = $2
			returning id, created_at, updated_at, project_id, kind, title, status, subtitle, payload, file_path
		`
		args = []any{userID, projectID, kind, title, status, subtitle, payloadRaw, filePath}
	}
	err = s.db.QueryRow(ctx, query, args...).Scan(
		&material.ID,
		&createdAt,
		&updatedAt,
		&material.ProjectID,
		&material.Kind,
		&material.Title,
		&material.Status,
		&material.Subtitle,
		&storedPayload,
		&material.FilePath,
	)
	if err != nil {
		return nil, fmt.Errorf("create material: %w", err)
	}
	material.CreatedAt = scanTimestampRFC3339(createdAt)
	material.UpdatedAt = scanTimestampRFC3339(updatedAt)
	if len(storedPayload) > 0 {
		if err := json.Unmarshal(storedPayload, &material.Payload); err != nil {
			return nil, fmt.Errorf("decode material payload: %w", err)
		}
	}
	if material.Payload == nil {
		material.Payload = map[string]any{}
	}
	return &material, nil
}

func (s *Store) GetMaterialByID(ctx context.Context, materialID int64) (*Material, error) {
	return s.GetMaterialByIDForUser(ctx, 0, materialID)
}

func (s *Store) GetMaterialByIDForUser(ctx context.Context, userID int64, materialID int64) (*Material, error) {
	var (
		material             Material
		createdAt, updatedAt time.Time
		payloadRaw           []byte
	)
	query := `
		select m.id, m.created_at, m.updated_at, m.project_id, m.kind, m.title, m.status, m.subtitle, m.payload, m.file_path
		from notex_materials m
		where m.id = $1
	`
	args := []any{materialID}
	if userID > 0 {
		query = `
			select m.id, m.created_at, m.updated_at, m.project_id, m.kind, m.title, m.status, m.subtitle, m.payload, m.file_path
			from notex_materials m
			join notex_projects p on p.id = m.project_id
			where p.user_id = $1 and m.id = $2
		`
		args = []any{userID, materialID}
	}
	err := s.db.QueryRow(ctx, query, args...).Scan(&material.ID, &createdAt, &updatedAt, &material.ProjectID, &material.Kind, &material.Title, &material.Status, &material.Subtitle, &payloadRaw, &material.FilePath)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get material: %w", err)
	}
	material.CreatedAt = scanTimestampRFC3339(createdAt)
	material.UpdatedAt = scanTimestampRFC3339(updatedAt)
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &material.Payload); err != nil {
			return nil, fmt.Errorf("decode material payload: %w", err)
		}
	}
	if material.Payload == nil {
		material.Payload = map[string]any{}
	}
	return &material, nil
}

// MarkAbandonedStudioPendingFailed marks pending/processing materials as failed when updated_at is before cutoff
// (e.g. user refreshed while generation was in flight; or studio/create left rows stuck in processing).
func (s *Store) MarkAbandonedStudioPendingFailed(ctx context.Context, userID int64, projectID string, cutoff time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("notex store is not initialized")
	}
	subtitle := "生成已中断（页面刷新或连接断开）；请重试"
	_, err := s.db.Exec(ctx, `
		update notex_materials m
		set status = 'failed', subtitle = $3, updated_at = now()
		from notex_projects p
		where p.user_id = $1 and p.id = $2 and m.project_id = p.id
		and m.status in ('pending', 'processing')
		and m.updated_at < $4
	`, userID, projectID, subtitle, cutoff)
	if err != nil {
		return fmt.Errorf("mark abandoned pending materials: %w", err)
	}
	return nil
}

// UpdateMaterialForUser updates an existing material row when the project is owned by userID.
func (s *Store) UpdateMaterialForUser(ctx context.Context, userID int64, projectID string, materialID int64, kind, title, status, subtitle string, payload map[string]any, filePath string) (*Material, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("notex store is not initialized")
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal material payload: %w", err)
	}
	var (
		material             Material
		createdAt, updatedAt time.Time
		storedPayload        []byte
	)
	err = s.db.QueryRow(ctx, `
		update notex_materials m
		set kind = $4, title = $5, status = $6, subtitle = $7, payload = $8, file_path = $9, updated_at = now()
		from notex_projects p
		where p.user_id = $1 and p.id = $2 and m.id = $3 and m.project_id = p.id
		returning m.id, m.created_at, m.updated_at, m.project_id, m.kind, m.title, m.status, m.subtitle, m.payload, m.file_path
	`, userID, projectID, materialID, kind, title, status, subtitle, payloadRaw, filePath).Scan(
		&material.ID,
		&createdAt,
		&updatedAt,
		&material.ProjectID,
		&material.Kind,
		&material.Title,
		&material.Status,
		&material.Subtitle,
		&storedPayload,
		&material.FilePath,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("update material: %w", err)
	}
	material.CreatedAt = scanTimestampRFC3339(createdAt)
	material.UpdatedAt = scanTimestampRFC3339(updatedAt)
	if len(storedPayload) > 0 {
		if err := json.Unmarshal(storedPayload, &material.Payload); err != nil {
			return nil, fmt.Errorf("decode material payload: %w", err)
		}
	}
	if material.Payload == nil {
		material.Payload = map[string]any{}
	}
	return &material, nil
}