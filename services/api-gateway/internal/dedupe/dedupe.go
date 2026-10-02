package dedupe

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	Client *redis.Client
	TTL    time.Duration
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	return &Store{
		Client: client,
		TTL:    ttl,
	}
}

func (s *Store) key(nodeID string, timestamp time.Time) string {
	return fmt.Sprintf(
		"sentinel:dedupe:%s:%s",
		nodeID,
		timestamp.UTC().Format(time.RFC3339Nano),
	)
}

func (s *Store) Exists(
	ctx context.Context,
	nodeID string,
	timestamp time.Time,
) (bool, error) {
	key := s.key(nodeID, timestamp)

	count, err := s.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) Mark(
	ctx context.Context,
	nodeID string,
	timestamp time.Time,
) error {
	key := s.key(nodeID, timestamp)

	return s.Client.Set(
		ctx,
		key,
		"1",
		s.TTL,
	).Err()
}