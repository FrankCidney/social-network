package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsWhenBucketIsEmpty(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, requestFrom("192.0.2.1"))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i+1, recorder.Code, http.StatusNoContent)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestFrom("192.0.2.1"))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header was not set")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(1, time.Second)
	limiter.now = func() time.Time { return now }

	if allowed, _ := limiter.allow("192.0.2.2"); !allowed {
		t.Fatal("first request was rejected")
	}
	if allowed, _ := limiter.allow("192.0.2.2"); allowed {
		t.Fatal("second immediate request was allowed")
	}

	now = now.Add(time.Second)
	if allowed, _ := limiter.allow("192.0.2.2"); !allowed {
		t.Fatal("request after refill was rejected")
	}
}

func TestRateLimiterSkipsOptions(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	calls := 0
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 3; i++ {
		req := requestFrom("192.0.2.3")
		req.Method = http.MethodOptions
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("OPTIONS status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
	}

	if calls != 3 {
		t.Fatalf("handler calls = %d, want 3", calls)
	}
}

func requestFrom(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = ip + ":12345"
	return req
}
