package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenRepo implements domain.TokenRepository using Redis.
type RedisTokenRepo struct {
	client *redis.Client
}

// NewRedisTokenRepo creates a new repository instance.
func NewRedisTokenRepo(client *redis.Client) *RedisTokenRepo {
	return &RedisTokenRepo{client: client}
}

// StoreRefreshToken saves an opaque token in Redis with a specific Time-To-Live (TTL).
// The key pattern is "auth:refresh:<token>" -> value "userID".
func (r *RedisTokenRepo) StoreRefreshToken(ctx context.Context, userID string, token string, ttl time.Duration) error {
	key := fmt.Sprintf("auth:refresh:%s", token)
	
	// We store the userID as the value so we can identify who owns the token later.
	err := r.client.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store token in redis: %w", err)
	}
	
	return nil
}

// GetUserIDByRefreshToken validates if a refresh token exists and returns the associated User ID.
func (r *RedisTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("auth:refresh:%s", token)
	
	userID, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("refresh token expired or invalid")
		}
		return "", fmt.Errorf("redis error: %w", err)
	}
	
	return userID, nil
}

// DeleteRefreshToken removes a token immediately.
// This is used for "Logout" or when a token is rotated.
func (r *RedisTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("auth:refresh:%s", token)
	return r.client.Del(ctx, key).Err()
}