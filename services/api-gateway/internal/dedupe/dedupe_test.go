package dedupe

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestStore(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running")
	}

	store := NewStore(client, time.Minute)

	nodeID := "test-node"
	timestamp := time.Now().UTC().Truncate(time.Second)

	exists, err := store.Exists(ctx, nodeID, timestamp)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Fatal("event should not exist initially")
	}

	if err := store.Mark(ctx, nodeID, timestamp); err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	exists, err = store.Exists(ctx, nodeID, timestamp)
	if err != nil {
		t.Fatalf("Exists failed after Mark: %v", err)
	}

	if !exists {
		t.Fatal("event should exist after Mark")
	}

	// Cleanup test key.
	key := store.key(nodeID, timestamp)
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}