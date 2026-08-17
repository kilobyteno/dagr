package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var nonHandle = regexp.MustCompile(`[^a-z0-9]+`)

// BaseHandle derives a handle candidate from a display name.
func BaseHandle(displayName string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '_'
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(displayName))
	handle := nonHandle.ReplaceAllString(lower, "_")
	handle = strings.Trim(handle, "_")
	for strings.Contains(handle, "__") {
		handle = strings.ReplaceAll(handle, "__", "_")
	}
	if handle == "" {
		handle = "member"
	}
	if len(handle) > 32 {
		handle = strings.Trim(handle[:32], "_")
	}
	if handle == "" {
		handle = "member"
	}
	return handle
}

type handleQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func allocateUniqueHandle(
	ctx context.Context,
	q handleQuerier,
	workspaceID uuid.UUID,
	displayName string,
) (string, error) {
	base := BaseHandle(displayName)
	candidate := base
	for i := 0; i < 50; i++ {
		var exists bool
		err := q.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM workspace_members
				WHERE workspace_id = $1 AND handle = $2
			)
		`, workspaceID, candidate).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("check handle: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s_%d", base, i+2)
		if len(candidate) > 32 {
			suffix := fmt.Sprintf("_%d", i+2)
			trim := 32 - len(suffix)
			if trim < 1 {
				trim = 1
			}
			candidate = base[:trim] + suffix
		}
	}
	return "", fmt.Errorf("could not allocate handle")
}
