package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(60, false) // 1 token/second, burst 60
	l.now = func() time.Time { return now }

	for i := 0; i < 60; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait != time.Second {
		t.Fatalf("61st request: ok=%v wait=%v, want false 1s", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other clients have their own bucket")
	}

	now = now.Add(time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a token should refill after a second")
	}

	// Full buckets are swept after a minute.
	now = now.Add(2 * time.Minute)
	l.Allow("c")
	if _, exists := l.buckets["a"]; exists {
		t.Error("idle bucket should have been swept")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, tc := range []struct {
		trustProxy bool
		wantSecond int
	}{
		{trustProxy: false, wantSecond: http.StatusTooManyRequests}, // same RemoteAddr
		{trustProxy: true, wantSecond: http.StatusOK},               // different forwarded IPs
	} {
		h := NewRateLimiter(1, tc.trustProxy).Middleware(ok)
		codes := []int{}
		for _, ip := range []string{"1.1.1.1", "2.2.2.2"} {
			req := httptest.NewRequest("POST", "/", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			req.Header.Set("X-Forwarded-For", ip+", 10.0.0.1")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			codes = append(codes, rec.Code)
		}
		if codes[0] != http.StatusOK || codes[1] != tc.wantSecond {
			t.Errorf("trustProxy=%v: codes %v, want [200 %d]", tc.trustProxy, codes, tc.wantSecond)
		}
	}
}

func TestCORS(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := CORS([]string{"https://app.example.com"})(next)

	// Preflight from an allowed origin is answered directly.
	req := httptest.NewRequest("OPTIONS", "/api/v1/items", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("preflight: %d %v", rec.Code, rec.Header())
	}

	// Disallowed origins get no CORS headers.
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin: %d %v", rec.Code, rec.Header())
	}

	// Wildcard.
	rec = httptest.NewRecorder()
	CORS([]string{"*"})(next).ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://evil.example.com" {
		t.Errorf("wildcard: %v", rec.Header())
	}
}
