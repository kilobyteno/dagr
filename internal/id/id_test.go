package id_test

import (
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/id"
)

func TestNewIsVersion7(t *testing.T) {
	t.Parallel()
	u := id.New()
	if v := u.Version(); v != 7 {
		t.Fatalf("version = %d, want 7", v)
	}
}
