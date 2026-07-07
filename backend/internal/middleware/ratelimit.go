package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"social-network/internal/apperror"
	"social-network/internal/response"
)

type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*rateBucket
	limit      int
	refillRate float64
	idleTTL    time.Duration
	lastPrune  time.Time
	now        func() time.Time
}

type rateBucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiter{
		buckets:    make(map[string]*rateBucket),
		limit:      limit,
		refillRate: float64(limit) / window.Seconds(),
		idleTTL:    5 * window,
		now:        time.Now,
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		allowed, retryAfter := l.allow(clientIP(r))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			response.Error(w, apperror.RateLimited("too many requests, please try again shortly"), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(key string) (bool, time.Duration) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastPrune) > time.Minute {
		l.prune(now)
		l.lastPrune = now
	}

	bucket := l.buckets[key]
	if bucket == nil {
		l.buckets[key] = &rateBucket{
			tokens:   float64(l.limit - 1),
			updated:  now,
			lastSeen: now,
		}
		return true, 0
	}

	elapsed := now.Sub(bucket.updated).Seconds()
	if elapsed > 0 {
		bucket.tokens = min(float64(l.limit), bucket.tokens+elapsed*l.refillRate)
		bucket.updated = now
	}
	bucket.lastSeen = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}

	waitSeconds := (1 - bucket.tokens) / l.refillRate
	return false, time.Duration(waitSeconds * float64(time.Second))
}

func (l *RateLimiter) prune(now time.Time) {
	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastSeen) > l.idleTTL {
			delete(l.buckets, key)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}

	return r.RemoteAddr
}
