// Package id generates time-ordered UUIDv7 identifiers.
package id

import "github.com/google/uuid"

// New returns a new UUIDv7, or panics if generation fails.
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
