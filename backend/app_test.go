package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeUpstream struct {
	calls int
	body  string
	err   error
	last  string
}

func (f *fakeUpstream) Range(_ context.Context, p string) (string, error) {
	f.calls++
	f.last = p
	return f.body, f.err
}

func newTestApp(up *fakeUpstream) (*App, *MemStore) {
	st := NewMemStore()
	return &App{Store: st, Upstream: up, CacheTTL: time.Hour, Limiter: NewRateLimiter(100, time.Minute)}, st
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	return rr
}

func TestRangeValidatesPrefix(t *testing.T) {
	app, _ := newTestApp(&fakeUpstream{body: "x"})
	h := app.Routes(t.TempDir())
	for _, p := range []string{"ABC", "ABCDEF", "GGGGG", "5BAA6%2F"} {
		if rr := get(t, h, "/api/range/"+p); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d want 400", p, rr.Code)
		}
	}
}

func TestRangeCachesAndUppercases(t *testing.T) {
	up := &fakeUpstream{body: "1E4C9B93F3F0682250B6CF8331B7EE68FD8:10434004\r\n"}
	app, _ := newTestApp(up)
	h := app.Routes(t.TempDir())

	rr := get(t, h, "/api/range/5baa6")
	if rr.Code != 200 || rr.Header().Get("X-Cache") != "MISS" || up.last != "5BAA6" {
		t.Fatalf("first: code=%d cache=%s last=%s", rr.Code, rr.Header().Get("X-Cache"), up.last)
	}
	if !strings.Contains(rr.Body.String(), "1E4C9B93") {
		t.Fatalf("body not proxied")
	}
	rr = get(t, h, "/api/range/5BAA6")
	if rr.Header().Get("X-Cache") != "HIT" || up.calls != 1 {
		t.Fatalf("second: cache=%s calls=%d", rr.Header().Get("X-Cache"), up.calls)
	}
}

func TestRangeServesStaleOnUpstreamFailure(t *testing.T) {
	up := &fakeUpstream{body: "AAAA:1\r\n"}
	app, st := newTestApp(up)
	app.CacheTTL = 0 // everything is stale
	h := app.Routes(t.TempDir())
	get(t, h, "/api/range/00000")
	up.err = errors.New("down")
	rr := get(t, h, "/api/range/00000")
	if rr.Code != 200 || rr.Header().Get("X-Cache") != "STALE" {
		t.Fatalf("got %d %s", rr.Code, rr.Header().Get("X-Cache"))
	}
	_ = st
	if rr := get(t, h, "/api/range/11111"); rr.Code != http.StatusBadGateway {
		t.Fatalf("uncached + down: got %d want 502", rr.Code)
	}
}

func TestRateLimit(t *testing.T) {
	app, _ := newTestApp(&fakeUpstream{body: "x"})
	app.Limiter = NewRateLimiter(2, time.Minute)
	h := app.Routes(t.TempDir())
	get(t, h, "/api/range/00000")
	get(t, h, "/api/range/00000")
	if rr := get(t, h, "/api/range/00000"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d want 429", rr.Code)
	}
}

func TestSecurityHeadersAndUnknownAPI(t *testing.T) {
	app, _ := newTestApp(&fakeUpstream{})
	h := app.Routes(t.TempDir())
	rr := get(t, h, "/api/nope")
	if rr.Code != 404 {
		t.Fatalf("got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "connect-src 'self'") {
		t.Fatal("missing CSP")
	}
}
