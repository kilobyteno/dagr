package presence

import (
	"context"
	"testing"
	"time"
)

func TestMemoryTouchGetAndClear(t *testing.T) {
	t.Parallel()
	store := NewMemory(time.Minute)
	ctx := context.Background()

	if got := store.Get(ctx, "user-1"); got != Offline {
		t.Fatalf("empty get = %s, want offline", got)
	}
	if err := store.Touch(ctx, "user-1", false); err != nil {
		t.Fatal(err)
	}
	if got := store.Get(ctx, "user-1"); got != Online {
		t.Fatalf("after touch = %s, want online", got)
	}
	if err := store.Touch(ctx, "user-1", true); err != nil {
		t.Fatal(err)
	}
	if got := store.Get(ctx, "user-1"); got != Away {
		t.Fatalf("after away = %s, want away", got)
	}
	if err := store.Clear(ctx, "user-1"); err != nil {
		t.Fatal(err)
	}
	if got := store.Get(ctx, "user-1"); got != Offline {
		t.Fatalf("after clear = %s, want offline", got)
	}
}

func TestMemoryExpiresToOffline(t *testing.T) {
	t.Parallel()
	store := NewMemory(20 * time.Millisecond)
	ctx := context.Background()
	if err := store.Touch(ctx, "user-2", false); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if got := store.Get(ctx, "user-2"); got != Offline {
		t.Fatalf("expired get = %s, want offline", got)
	}
}

func TestMemoryGetMany(t *testing.T) {
	t.Parallel()
	store := NewMemory(time.Minute)
	ctx := context.Background()
	_ = store.Touch(ctx, "online-user", false)
	_ = store.Touch(ctx, "away-user", true)

	got := store.GetMany(ctx, []string{"online-user", "away-user", "missing"})
	if got["online-user"] != Online || got["away-user"] != Away || got["missing"] != Offline {
		t.Fatalf("GetMany = %+v", got)
	}
}

func TestMemoryIgnoresBlankUser(t *testing.T) {
	t.Parallel()
	store := NewMemory(time.Minute)
	ctx := context.Background()
	if err := store.Touch(ctx, "  ", false); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if len(store.GetMany(ctx, nil)) != 0 {
		t.Fatal("expected empty GetMany")
	}
}

func TestRedisNilClientIsOffline(t *testing.T) {
	t.Parallel()
	store := NewRedis(nil, 0)
	ctx := context.Background()
	if err := store.Touch(ctx, "user-1", false); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear(ctx, "user-1"); err != nil {
		t.Fatal(err)
	}
	if got := store.Get(ctx, "user-1"); got != Offline {
		t.Fatalf("nil redis get = %s, want offline", got)
	}
	got := store.GetMany(ctx, []string{"user-1"})
	if got["user-1"] != Offline {
		t.Fatalf("nil redis GetMany = %+v", got)
	}
}
