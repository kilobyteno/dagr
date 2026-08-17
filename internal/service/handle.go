package service

import (
	"errors"
	"regexp"

	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrHandleTaken   = errors.New("handle already taken")
	ErrInvalidHandle = errors.New("invalid handle")
	handlePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{1,31}$`)
)

// NormaliseHandle validates and normalises a workspace handle.
func NormaliseHandle(raw string) (string, error) {
	handle := postgres.BaseHandle(raw)
	if !handlePattern.MatchString(handle) {
		return "", ErrInvalidHandle
	}
	return handle, nil
}
