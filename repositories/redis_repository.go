package repositories

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

type redisRepositoryImpl struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) RedisRepository {
	return &redisRepositoryImpl{client: client}
}

func (r *redisRepositoryImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *redisRepositoryImpl) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisRepositoryImpl) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
