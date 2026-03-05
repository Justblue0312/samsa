package middleware

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	ratelimiter "github.com/nccapo/rate-limiter"
	"github.com/redis/go-redis/v9"
)

type LimiterStrategy string

var (
	SlidingWindow LimiterStrategy = "sliding"
	FixedWindow   LimiterStrategy = "fixed"
)

type Rule struct {
	RateLimitGroup sqlc.RateLimitGroup
	Pattern        string
	Limit          int
	Window         time.Duration
	Strategy       LimiterStrategy
}

func RateLimiter(rdb *redis.Client, rules []Rule) func(http.Handler) http.Handler {
	type compiledRule struct {
		matcher func(string) bool
		limiter *ratelimiter.RateLimiter
	}

	store := ratelimiter.NewRedisStore(rdb, true)

	// Index rules by group — O(1) group lookup instead of scanning all rules
	rulesByGroup := make(map[sqlc.RateLimitGroup][]compiledRule)
	for _, rule := range rules {
		limiter, err := ratelimiter.NewRateLimiter(
			ratelimiter.WithRate(1),
			ratelimiter.WithMaxTokens(int64(rule.Limit)),
			ratelimiter.WithRefillInterval(rule.Window),
			ratelimiter.WithStore(store),
		)
		if err != nil {
			log.Fatalf("failed to create limiter: %v", err)
		}

		rulesByGroup[rule.RateLimitGroup] = append(rulesByGroup[rule.RateLimitGroup], compiledRule{
			matcher: buildMatcher(rule.Pattern),
			limiter: limiter,
		})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var rlGroup sqlc.RateLimitGroup
			sub, err := common.GetAuthSubject(ctx)
			if err != nil || sub.IsAnonymous() {
				rlGroup = sqlc.RateLimitGroupDefault
			} else {
				rlGroup = sub.User.RateLimitGroup
			}

			// Only iterate rules for this group
			if groupRules, ok := rulesByGroup[rlGroup]; ok {
				path := r.URL.Path
				ip := clientIP(r) // resolve once, reuse across rule checks
				for _, rule := range groupRules {
					if rule.matcher(path) {
						result, err := rule.limiter.Allow(ctx, ip)
						if err != nil {
							slog.Info("rate limiter error", "error", err.Error())
							break
						}
						if !result.Allowed {
							http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
							return
						}
						break
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func buildMatcher(pattern string) func(string) bool {
	// Avoid regex entirely for patterns without wildcards
	if !strings.Contains(pattern, "*") {
		exact := "/" + pattern
		return func(path string) bool { return path == exact }
	}

	// For prefix patterns like "foo/bar*", use HasPrefix — much faster than regex
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		prefix := "/" + pattern[:len(pattern)-1]
		return func(path string) bool { return strings.HasPrefix(path, prefix) }
	}

	// Fall back to regex only when necessary (multiple or mid-string wildcards)
	re := regexp.MustCompile("^/" + strings.ReplaceAll(pattern, "*", ".*"))
	return re.MatchString
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// strings.Cut avoids allocating a slice just to take index [0]
		if before, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(before)
		}
		return strings.TrimSpace(xff)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
