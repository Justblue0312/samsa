package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/justblue/samsa/gen/sqlc"
	"github.com/redis/go-redis/v9"
)

func GetStateKey(nonce string, provider sqlc.OAuthProvider) string {
	return fmt.Sprintf("oauth_state:%s:%s", provider, nonce)
}

func StoreState(ctx context.Context, rdb *redis.Client, nonce string, provider sqlc.OAuthProvider, stateData map[string]any, ttl time.Duration) error {
	key := GetStateKey(nonce, provider)
	data, err := json.Marshal(stateData)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, data, ttl).Err()
}

func RetrieveState(ctx context.Context, rdb *redis.Client, nonce string, provider sqlc.OAuthProvider) (map[string]any, error) {
	key := GetStateKey(nonce, provider)
	stateJSON, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var stateData map[string]any
	if err = json.Unmarshal([]byte(stateJSON), &stateData); err != nil {
		return nil, err
	}
	return stateData, nil
}

func DeleteState(ctx context.Context, rdb *redis.Client, nonce string, provider sqlc.OAuthProvider) error {
	key := GetStateKey(nonce, provider)
	return rdb.Del(ctx, key).Err()
}

// IsSecureRequest returns true when the request does not appear to come from localhost.
func IsSecureRequest(r *http.Request) bool {
	hostname := r.URL.Hostname()
	return !slices.Contains([]string{"localhost", "127.0.0.1"}, hostname)
}

func SetLoginCookie(w http.ResponseWriter, r *http.Request, cookieName string, nonce string, ttl time.Duration, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    nonce,
		MaxAge:   int(ttl.Seconds()),
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearLoginCookie(w http.ResponseWriter, r *http.Request, cookieName string, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
