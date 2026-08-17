// Package repository defines persistence interfaces and will host pgx/sqlc adapters.
package repository

import (
	"context"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

// UserStore persists users.
type UserStore interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// Auth persistence lives in repository/postgres for the current vertical slice.

// ChannelStore persists channels.
type ChannelStore interface {
	GetByID(ctx context.Context, id string) (*domain.Channel, error)
}

// MessageStore persists messages.
type MessageStore interface {
	GetByID(ctx context.Context, id string) (*domain.Message, error)
}

// FileStore persists file attachment metadata.
type FileStore interface {
	GetByID(ctx context.Context, id string) (*domain.FileAttachment, error)
}

// NotImplemented is returned by stub adapters until sqlc queries exist.
type NotImplemented struct {
	Op string
}

func (e *NotImplemented) Error() string {
	if e.Op == "" {
		return "repository: not implemented"
	}
	return "repository: not implemented: " + e.Op
}
