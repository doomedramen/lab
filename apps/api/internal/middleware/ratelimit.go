package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

// RateLimiter implements IP-based rate limiting for Connect RPC handlers.
// Each unique IP gets its own token bucket. Entries expire after inactivity
// to prevent unbounded memory growth.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	// Rate: events per second, Burst: allowed burst above rate
	rps   rate.Limit
	burst int
	// ttl is how long an idle entry is retained before cleanup
	ttl time.Duration
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter that allows `rps` requests per second
// with a burst capacity of `burst` per IP address.
func NewRateLimiter(rps rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rps:      rps,
		burst:    burst,
		ttl:      10 * time.Minute,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &rateLimiterEntry{
			limiter: rate.NewLimiter(rl.rps, rl.burst),
		}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes stale entries to keep memory bounded.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.ttl)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.ttl)
		for ip, entry := range rl.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Interceptor returns a Connect unary interceptor that enforces rate limiting.
// Requests that exceed the limit receive a ResourceExhausted error with a
// Retry-After header indicating when the client can retry.
func (rl *RateLimiter) Interceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ip := extractIP(req.Header())
			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				// Calculate retry-after based on rate (1/rps seconds)
				retryAfter := time.Duration(float64(time.Second) / float64(rl.rps))
				err := connect.NewError(
					connect.CodeResourceExhausted,
					nil,
				)
				// Include retry-after in error metadata
				err.Meta().Set("Retry-After", retryAfter.Round(time.Second).String())
				return nil, err
			}
			return next(ctx, req)
		}
	}
}

// HTTPMiddleware returns an http.Handler middleware that enforces rate limiting.
// Use this for plain HTTP endpoints (not Connect RPC). Returns 429 with
// Retry-After header when rate limit is exceeded.
func (rl *RateLimiter) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIPFromRequest(r)
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			// Calculate retry-after based on rate (1/rps seconds)
			retryAfter := time.Duration(float64(time.Second) / float64(rl.rps))
			w.Header().Set("Retry-After", retryAfter.Round(time.Second).String())
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractIP reads the client IP from the request headers, preferring
// X-Forwarded-For (set by reverse proxies) over the direct remote address.
func extractIP(headers http.Header) string {
	// X-Real-IP (set by nginx) takes precedence
	if ip := headers.Get("X-Real-IP"); ip != "" {
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String()
		}
	}
	// X-Forwarded-For may contain a comma-separated list; use the first entry
	if xff := headers.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in the chain
		for _, part := range splitCSV(xff) {
			if parsed := net.ParseIP(part); parsed != nil {
				return parsed.String()
			}
		}
	}
	return "unknown"
}

func extractIPFromRequest(r *http.Request) string {
	ip := extractIP(r.Header)
	if ip != "unknown" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func splitCSV(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			trimmed := trimSpace(s[start:i])
			if trimmed != "" {
				result = append(result, trimmed)
			}
			start = i + 1
		}
	}
	if trimmed := trimSpace(s[start:]); trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
