package presence

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// State is a user's live availability.
type State string

const (
	Online  State = "online"
	Away    State = "away"
	Offline State = "offline"
)

const (
	keyPrefix   = "dagr:presence:"
	defaultTTL  = 90 * time.Second
	manualAway  = "away"
	activeValue = "active"
)

// Store tracks ephemeral online/away state.
type Store interface {
	Touch(ctx context.Context, userID string, away bool) error
	Get(ctx context.Context, userID string) State
	GetMany(ctx context.Context, userIDs []string) map[string]State
}

// Memory is an in-process presence store for tests and Redis-less runs.
type Memory struct {
	mu    sync.Mutex
	items map[string]memoryItem
	ttl   time.Duration
}

type memoryItem struct {
	away      bool
	expiresAt time.Time
}

func NewMemory(ttl time.Duration) *Memory {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &Memory{items: map[string]memoryItem{}, ttl: ttl}
}

func (m *Memory) Touch(_ context.Context, userID string, away bool) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[userID] = memoryItem{away: away, expiresAt: time.Now().Add(m.ttl)}
	return nil
}

func (m *Memory) Get(_ context.Context, userID string) State {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[userID]
	if !ok || time.Now().After(item.expiresAt) {
		delete(m.items, userID)
		return Offline
	}
	if item.away {
		return Away
	}
	return Online
}

func (m *Memory) GetMany(ctx context.Context, userIDs []string) map[string]State {
	out := make(map[string]State, len(userIDs))
	for _, id := range userIDs {
		out[id] = m.Get(ctx, id)
	}
	return out
}

// RedisStore stores presence keys with TTL in Redis.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(client *redis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &RedisStore{client: client, ttl: ttl}
}

func (r *RedisStore) Touch(ctx context.Context, userID string, away bool) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || r.client == nil {
		return nil
	}
	value := activeValue
	if away {
		value = manualAway
	}
	return r.client.Set(ctx, keyPrefix+userID, value, r.ttl).Err()
}

func (r *RedisStore) Get(ctx context.Context, userID string) State {
	if r.client == nil || strings.TrimSpace(userID) == "" {
		return Offline
	}
	value, err := r.client.Get(ctx, keyPrefix+userID).Result()
	if err != nil {
		return Offline
	}
	if value == manualAway {
		return Away
	}
	return Online
}

func (r *RedisStore) GetMany(ctx context.Context, userIDs []string) map[string]State {
	out := make(map[string]State, len(userIDs))
	if r.client == nil || len(userIDs) == 0 {
		for _, id := range userIDs {
			out[id] = Offline
		}
		return out
	}
	keys := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		keys = append(keys, keyPrefix+id)
	}
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		for _, id := range userIDs {
			out[id] = Offline
		}
		return out
	}
	for i, id := range userIDs {
		raw, _ := values[i].(string)
		switch raw {
		case manualAway:
			out[id] = Away
		case activeValue:
			out[id] = Online
		default:
			out[id] = Offline
		}
	}
	return out
}
