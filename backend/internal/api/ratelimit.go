package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter is a simple "N requests per minute per client IP" limiter.
// It uses a fixed window: each IP gets a counter that resets every minute.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limitPerMinute int) *rateLimiter {
	return &rateLimiter{
		limit:   limitPerMinute,
		window:  time.Minute,
		clients: make(map[string]*window),
	}
}

// allow returns true if this client may make another request now.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Keep memory under control: when the map gets big, drop expired entries.
	if len(rl.clients) > 10000 {
		for k, w := range rl.clients {
			if now.After(w.resetAt) {
				delete(rl.clients, k)
			}
		}
	}

	w, ok := rl.clients[key]
	if !ok || now.After(w.resetAt) {
		rl.clients[key] = &window{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}

// wrap protects a handler with the limiter. A limit of 0 or less disables it.
func (rl *rateLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	if rl.limit <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, time.Now(), "too many requests, please slow down")
			return
		}
		next(w, r)
	}
}

// clientIP prefers X-Forwarded-For because on platforms like Render or Railway
// the real client sits behind a proxy.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
