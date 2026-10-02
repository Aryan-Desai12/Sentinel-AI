package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	Client *redis.Client
	Burst  int
}

func NewLimiter(client *redis.Client, burst int) *Limiter {
	return &Limiter{
		Client: client,
		Burst:  burst,
	}
}

func (l *Limiter) Allow(ctx context.Context, apiKey string) (bool, error) {
	key := fmt.Sprintf("sentinel:ratelimit:%s", apiKey)

	count, err := l.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		if err := l.Client.Expire(ctx, key, time.Second).Err(); err != nil {
			return false, err
		}
	}

	if count > int64(l.Burst) {
		return false, nil
	}

	return true, nil
}
