package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestLimiter(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running")
	}

	apiKey := "test-api-key"

	// Clean up before the test.
	client.Del(ctx, "sentinel:ratelimit:"+apiKey)

	limiter := NewLimiter(client, 3)

	// First three requests should be allowed.
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, apiKey)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}

		if !allowed {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	// Fourth request should be rejected.
	allowed, err := limiter.Allow(ctx, apiKey)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}

	if allowed {
		t.Fatal("fourth request should have been rejected")
	}

	// Wait for the fixed window to expire.
	time.Sleep(1100 * time.Millisecond)

	// New request should be allowed.
	allowed, err = limiter.Allow(ctx, apiKey)
	if err != nil {
		t.Fatalf("Allow failed after window reset: %v", err)
	}

	if !allowed {
		t.Fatal("request should have been allowed after window reset")
	}

	client.Del(ctx, "sentinel:ratelimit:"+apiKey)
}
