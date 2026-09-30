package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yeabt/lewe/internal/response"
)

// RateLimiter is an in-memory token-bucket limiter keyed by client IP.
// Each client may make up to perMinute requests per minute, with bursts of
// the same size. State is per process, which suits a single API instance.
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	rate       float64 // tokens per second
	burst      float64
	trustProxy bool
	lastSweep  time.Time
	now        func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a limiter allowing perMinute requests per client per
// minute. When trustProxy is true the client IP is taken from the first
// X-Forwarded-For entry; only enable it behind a proxy that sets that header.
func NewRateLimiter(perMinute int, trustProxy bool) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*bucket),
		rate:       float64(perMinute) / 60,
		burst:      float64(perMinute),
		trustProxy: trustProxy,
		now:        time.Now,
	}
}

// Allow consumes a token for key. If none is available it returns false and
// how long until one will be.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep drops buckets that have refilled completely, bounding memory use.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, k)
		}
	}
}

// Middleware rejects requests over the limit with 429 Too Many Requests.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := l.Allow(l.clientIP(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			response.Error(w, http.StatusTooManyRequests, "Too many requests, please try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) clientIP(r *http.Request) string {
	if l.trustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			return strings.TrimSpace(strings.Split(fwd, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
