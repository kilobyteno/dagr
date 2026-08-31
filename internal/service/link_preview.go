package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/unfurl"
)

// LinkUnfurlEnqueuer schedules background URL unfurl work.
type LinkUnfurlEnqueuer interface {
	EnqueueLinkUnfurl(ctx context.Context, previewID string) error
}

type linkPreviewStore interface {
	InsertLinkPreview(ctx context.Context, in postgres.InsertLinkPreviewInput) (postgres.LinkPreviewRow, error)
	GetLinkPreview(ctx context.Context, id uuid.UUID) (postgres.LinkPreviewRow, error)
	ListLinkPreviewsForMessages(ctx context.Context, messageIDs []uuid.UUID) ([]postgres.LinkPreviewRow, error)
	ListPendingLinkPreviews(ctx context.Context, olderThan time.Duration, limit int) ([]postgres.LinkPreviewRow, error)
	UpdateLinkPreview(ctx context.Context, in postgres.UpdateLinkPreviewInput) error
}

func (s *MessageService) WithLinkUnfurl(enqueuer LinkUnfurlEnqueuer) *MessageService {
	s.unfurl = enqueuer
	return s
}

func (s *MessageService) queueLinkPreviews(ctx context.Context, messageID uuid.UUID, body, contentType string) {
	if contentType == domain.ContentTypeSystem || contentType == domain.ContentTypeRich {
		return
	}
	urls := unfurl.ExtractURLs(body)
	if len(urls) == 0 {
		return
	}
	store, ok := s.store.(linkPreviewStore)
	if !ok {
		return
	}
	for _, raw := range urls {
		row, err := store.InsertLinkPreview(ctx, postgres.InsertLinkPreviewInput{
			MessageID:     messageID,
			URL:           raw,
			NormalizedURL: unfurl.NormalizeURLString(raw),
		})
		if err != nil {
			continue
		}
		if row.Status != string(domain.LinkPreviewPending) {
			continue
		}
		s.scheduleLinkPreview(ctx, row.ID.String())
	}
}

func (s *MessageService) scheduleLinkPreview(ctx context.Context, previewID string) {
	if s.unfurl != nil {
		if err := s.unfurl.EnqueueLinkUnfurl(ctx, previewID); err == nil {
			return
		}
	}
	// Fall back to inline fetch when Redis is unavailable or enqueue fails.
	go func(id string) {
		bg, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = s.ProcessLinkPreview(bg, id)
	}(previewID)
}

func (s *MessageService) attachLinkPreviews(ctx context.Context, messages []domain.Message) []domain.Message {
	if len(messages) == 0 {
		return messages
	}
	store, ok := s.store.(linkPreviewStore)
	if !ok {
		return messages
	}
	ids := make([]uuid.UUID, 0, len(messages))
	index := map[uuid.UUID]int{}
	for i, msg := range messages {
		id, err := uuid.Parse(msg.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		index[id] = i
	}
	rows, err := store.ListLinkPreviewsForMessages(ctx, ids)
	if err != nil {
		return messages
	}
	for _, row := range rows {
		i, ok := index[row.MessageID]
		if !ok {
			continue
		}
		preview := row.ToDomain()
		// Only surface useful cards in the client payload.
		if preview.Status != domain.LinkPreviewReady && preview.Status != domain.LinkPreviewPending {
			continue
		}
		messages[i].LinkPreviews = append(messages[i].LinkPreviews, preview)
	}
	return messages
}

// ProcessLinkPreview fetches metadata for a pending preview row.
func (s *MessageService) ProcessLinkPreview(ctx context.Context, previewID string) error {
	store, ok := s.store.(linkPreviewStore)
	if !ok {
		return nil
	}
	id, err := uuid.Parse(previewID)
	if err != nil {
		return ErrInvalidInput
	}
	row, err := store.GetLinkPreview(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if row.Status != string(domain.LinkPreviewPending) {
		return nil
	}

	result, fetchErr := unfurl.Fetch(ctx, row.URL)
	finalURL := row.URL
	site := hostLabel(row.URL)
	title := site
	description := ""
	imageURL := ""
	errText := ""
	if fetchErr != nil {
		// Keep a basic card in chat even when the remote page cannot be fetched.
		errText = truncateErr(fetchErr.Error(), 240)
		if title == "" {
			title = row.URL
		}
	} else {
		if strings.TrimSpace(result.URL) != "" {
			finalURL = strings.TrimSpace(result.URL)
		}
		site = strings.TrimSpace(result.SiteName)
		if site == "" {
			site = hostLabel(finalURL)
		}
		title = strings.TrimSpace(result.Title)
		if title == "" {
			title = site
		}
		if title == "" {
			title = finalURL
		}
		description = result.Description
		imageURL = result.ImageURL
	}

	return store.UpdateLinkPreview(ctx, postgres.UpdateLinkPreviewInput{
		ID:          id,
		Status:      domain.LinkPreviewReady,
		Title:       title,
		Description: description,
		SiteName:    site,
		ImageURL:    imageURL,
		URL:         finalURL,
		Error:       errText,
	})
}

// ProcessPendingLinkPreviews reclaims stuck pending unfurls (for example after Redis blips).
func (s *MessageService) ProcessPendingLinkPreviews(ctx context.Context, limit int) (int, error) {
	store, ok := s.store.(linkPreviewStore)
	if !ok {
		return 0, nil
	}
	rows, err := store.ListPendingLinkPreviews(ctx, 5*time.Second, limit)
	if err != nil {
		return 0, err
	}
	done := 0
	for _, row := range rows {
		if err := s.ProcessLinkPreview(ctx, row.ID.String()); err != nil {
			continue
		}
		done++
	}
	return done, nil
}

func hostLabel(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	return host
}

func truncateErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
