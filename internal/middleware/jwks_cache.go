package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// JWKSCache caches the JWKS response from korauth in Redis.
type JWKSCache struct {
	rdb     *redis.Client
	authURL string
	ttl     time.Duration
}

// NewJWKSCache creates a new JWKSCache.
func NewJWKSCache(rdb *redis.Client, authURL string, ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		rdb:     rdb,
		authURL: authURL,
		ttl:     ttl,
	}
}

// GetJWKS returns the JWKS, served from cache when available.
func (jc *JWKSCache) GetJWKS(ctx context.Context) (*JWKS, error) {
	const cacheKey = "jwks"

	cached, err := jc.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var jwks JWKS
		if err := json.Unmarshal([]byte(cached), &jwks); err != nil {
			fmt.Printf("Failed to unmarshal cached JWKS: %v\n", err)
		} else {
			return &jwks, nil
		}
	} else if err != redis.Nil {
		fmt.Printf("Redis error: %v\n", err)
	}

	jwks, err := jc.fetchFromAuth()
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(jwks)
	if err := jc.rdb.Set(ctx, cacheKey, string(data), jc.ttl).Err(); err != nil {
		fmt.Printf("Failed to cache JWKS: %v\n", err)
	}

	return jwks, nil
}

// fetchFromAuth fetches the JWKS directly from korauth.
func (jc *JWKSCache) fetchFromAuth() (*JWKS, error) {
	url := fmt.Sprintf("%s/v1/auth/.well-known/jwks.json", jc.authURL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	return &jwks, nil
}
