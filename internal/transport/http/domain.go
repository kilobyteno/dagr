package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type addDomainRequest struct {
	Domain string `json:"domain"`
}

type patchDomainRequest struct {
	AutoJoin *bool `json:"autoJoin"`
}

type domainJSON struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspaceId"`
	Domain            string     `json:"domain"`
	Verified          bool       `json:"verified"`
	VerifiedAt        *time.Time `json:"verifiedAt,omitempty"`
	AutoJoin          bool       `json:"autoJoin"`
	VerificationToken string     `json:"verificationToken,omitempty"`
	DNSHost           string     `json:"dnsHost"`
	DNSType           string     `json:"dnsType"`
	DNSValue          string     `json:"dnsValue"`
}

type listDomainsResponse struct {
	Domains []domainJSON `json:"domains"`
}

type domainResponse struct {
	Domain domainJSON `json:"domain"`
}

func toDomainJSON(d domain.WorkspaceDomain) domainJSON {
	out := domainJSON{
		ID:          d.ID,
		WorkspaceID: d.WorkspaceID,
		Domain:      d.Domain,
		Verified:    d.Verified(),
		VerifiedAt:  d.VerifiedAt,
		AutoJoin:    d.AutoJoin,
		DNSHost:     service.DomainTXTHost(d.Domain),
		DNSType:     "TXT",
		DNSValue:    service.DomainTXTValue(d.VerificationToken),
	}
	if !d.Verified() {
		out.VerificationToken = d.VerificationToken
	}
	return out
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.domains.List(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]domainJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toDomainJSON(item))
	}
	writeJSON(w, http.StatusOK, listDomainsResponse{Domains: out})
}

func (s *Server) handleAddDomain(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req addDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	created, err := s.domains.Add(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Domain)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, domainResponse{Domain: toDomainJSON(*created)})
}

func (s *Server) handleVerifyDomain(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	verified, err := s.domains.Verify(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "domainID"),
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, domainResponse{Domain: toDomainJSON(*verified)})
}

func (s *Server) handlePatchDomain(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req patchDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if req.AutoJoin == nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "autoJoin is required")
		return
	}
	updated, err := s.domains.SetAutoJoin(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "domainID"),
		*req.AutoJoin,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, domainResponse{Domain: toDomainJSON(*updated)})
}

func (s *Server) handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.domains.Delete(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "domainID"),
	); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid domain")
	case errors.Is(err, service.ErrDomainDenied):
		writeError(w, http.StatusBadRequest, "domain_denied", "Public email domains cannot be verified")
	case errors.Is(err, service.ErrDomainConflict):
		writeError(w, http.StatusConflict, "domain_conflict", "Domain is already claimed")
	case errors.Is(err, service.ErrDomainDNSMismatch):
		writeError(w, http.StatusBadRequest, "dns_mismatch", "DNS TXT verification record not found")
	case errors.Is(err, service.ErrDomainUnverified):
		writeError(w, http.StatusBadRequest, "domain_unverified", "Verify the domain before enabling auto-join")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Domain not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission to manage this workspace")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	}
}
