package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type createDocumentRequest struct {
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	Slug     string  `json:"slug"`
	Icon     string  `json:"icon"`
	ParentID *string `json:"parentId"`
}

type updateDocumentRequest struct {
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	Slug     *string `json:"slug"`
	Icon     *string `json:"icon"`
	ParentID *string `json:"parentId"`
}

type documentJSON struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	ParentID    string    `json:"parentId,omitempty"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Body        *string   `json:"body,omitempty"`
	Icon        string    `json:"icon"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	UpdatedBy   string    `json:"updatedBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type documentRevisionJSON struct {
	ID            string    `json:"id"`
	DocumentID    string    `json:"documentId"`
	Version       int       `json:"version"`
	ParentID      string    `json:"parentId,omitempty"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Body          *string   `json:"body,omitempty"`
	Icon          string    `json:"icon"`
	CreatedBy     string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type listDocumentsResponse struct {
	Documents []documentJSON `json:"documents"`
}

func toDocumentJSON(doc domain.Document, includeBody bool) documentJSON {
	out := documentJSON{
		ID:          doc.ID,
		WorkspaceID: doc.WorkspaceID,
		ParentID:    doc.ParentID,
		Slug:        doc.Slug,
		Title:       doc.Title,
		Icon:        doc.Icon,
		CreatedBy:   doc.CreatedBy,
		UpdatedBy:   doc.UpdatedBy,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
	if includeBody {
		body := doc.Body
		out.Body = &body
	}
	return out
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.documents.List(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	out := make([]documentJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toDocumentJSON(item, false))
	}
	writeJSON(w, http.StatusOK, listDocumentsResponse{Documents: out})
}

func (s *Server) handleSearchDocuments(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.documents.Search(
		r.Context(), user.ID, chi.URLParam(r, "workspaceID"), strings.TrimSpace(r.URL.Query().Get("q")),
	)
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	out := make([]documentJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toDocumentJSON(item, false))
	}
	writeJSON(w, http.StatusOK, listDocumentsResponse{Documents: out})
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req createDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	parentID := ""
	if req.ParentID != nil {
		parentID = *req.ParentID
	}
	doc, err := s.documents.Create(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), service.CreateDocumentInput{
		Title:    req.Title,
		Body:     req.Body,
		Slug:     req.Slug,
		Icon:     req.Icon,
		ParentID: parentID,
	})
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": toDocumentJSON(*doc, true)})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	doc, err := s.documents.Get(r.Context(), user.ID, chi.URLParam(r, "documentID"))
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": toDocumentJSON(*doc, true)})
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req updateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	doc, err := s.documents.Update(r.Context(), user.ID, chi.URLParam(r, "documentID"), service.UpdateDocumentInput{
		Title:    req.Title,
		Body:     req.Body,
		Slug:     req.Slug,
		Icon:     req.Icon,
		ParentID: req.ParentID,
	})
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": toDocumentJSON(*doc, true)})
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.documents.Delete(r.Context(), user.ID, chi.URLParam(r, "documentID")); err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDocumentRevisions(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.documents.ListRevisions(r.Context(), user.ID, chi.URLParam(r, "documentID"))
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	out := make([]documentRevisionJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toDocumentRevisionJSON(item, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{"revisions": out})
}

func (s *Server) handleGetDocumentRevision(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	rev, err := s.documents.GetRevision(
		r.Context(), user.ID, chi.URLParam(r, "documentID"), chi.URLParam(r, "revisionID"),
	)
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": toDocumentRevisionJSON(*rev, true)})
}

func (s *Server) handleRestoreDocumentRevision(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.documents == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	doc, err := s.documents.RestoreRevision(
		r.Context(), user.ID, chi.URLParam(r, "documentID"), chi.URLParam(r, "revisionID"),
	)
	if err != nil {
		s.writeDocumentError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": toDocumentJSON(*doc, true)})
}

func toDocumentRevisionJSON(rev domain.DocumentRevision, includeBody bool) documentRevisionJSON {
	out := documentRevisionJSON{
		ID:            rev.ID,
		DocumentID:    rev.DocumentID,
		Version:       rev.Version,
		ParentID:      rev.ParentID,
		Slug:          rev.Slug,
		Title:         rev.Title,
		Icon:          rev.Icon,
		CreatedBy:     rev.CreatedBy,
		CreatedByName: rev.CreatedByName,
		CreatedAt:     rev.CreatedAt,
	}
	if includeBody {
		body := rev.Body
		out.Body = &body
	}
	return out
}

func (s *Server) writeDocumentError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrDocumentSlug), errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid document input", err)
	case errors.Is(err, service.ErrDocumentCycle):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "A page cannot be moved under itself", err)
	case errors.Is(err, service.ErrDocumentDepth):
		s.writeError(w, r, http.StatusBadRequest, "nesting_too_deep", "Pages can nest at most five levels under a top level page", err)
	case errors.Is(err, service.ErrDocumentHasChildren):
		s.writeError(w, r, http.StatusConflict, "has_children", "Move or delete child pages first", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Document not found", err)
	case errors.Is(err, service.ErrForbidden):
		s.writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Could not update documents", err)
	}
}
