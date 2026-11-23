package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Store persists cached range responses and anonymous aggregate counters.
// It never receives passwords or full hashes.
type Store interface {
	GetRange(ctx context.Context, prefix string) (body string, fetchedAt time.Time, ok bool, err error)
	PutRange(ctx context.Context, prefix, body string) error
	IncrementChecks(ctx context.Context) error
	TotalChecks(ctx context.Context) (int64, error)
	Ping(ctx context.Context) error
	Close()
}

// Upstream fetches a range response from Have I Been Pwned.
type Upstream interface {
	Range(ctx context.Context, prefix string) (string, error)
}

type App struct {
	Store    Store
	Upstream Upstream
	CacheTTL time.Duration
	Limiter  *RateLimiter
}

var prefixRe = regexp.MustCompile(`^[0-9A-F]{5}$`)

func (a *App) Routes(staticDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/range/{prefix}", a.handleRange)
	mux.HandleFunc("GET /api/stats", a.handleStats)
	mux.HandleFunc("GET /healthz", a.handleHealth)
	mux.Handle("/", spaHandler(staticDir))
	return securityHeaders(mux)
}

func (a *App) handleRange(w http.ResponseWriter, r *http.Request) {
	if !a.Limiter.Allow(clientIP(r)) {
		w.Header().Set("Retry-After", "60")
		httpError(w, http.StatusTooManyRequests, "too many requests, slow down")
		return
	}

	prefix := strings.ToUpper(r.PathValue("prefix"))
	if !prefixRe.MatchString(prefix) {
		httpError(w, http.StatusBadRequest, "prefix must be exactly 5 hex characters")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	body, source, err := a.lookup(ctx, prefix)
	if err != nil {
		log.Printf("range lookup failed: %v", err)
		httpError(w, http.StatusBadGateway, "breach database is unreachable right now, try again shortly")
		return
	}

	// Count the check (no prefix, no IP, nothing identifying), best effort.
	go func() {
		c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := a.Store.IncrementChecks(c); err != nil {
			log.Printf("increment checks: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Cache", source)
	_, _ = w.Write([]byte(body))
}

// lookup returns the range body from cache when fresh, otherwise from upstream.
// If upstream fails but a stale cached copy exists, the stale copy is served.
func (a *App) lookup(ctx context.Context, prefix string) (string, string, error) {
	cached, fetchedAt, ok, err := a.Store.GetRange(ctx, prefix)
	if err != nil {
		log.Printf("cache read: %v", err)
		ok = false
	}
	if ok && time.Since(fetchedAt) < a.CacheTTL {
		return cached, "HIT", nil
	}

	body, upErr := a.Upstream.Range(ctx, prefix)
	if upErr != nil {
		if ok {
			return cached, "STALE", nil
		}
		return "", "", upErr
	}
	if err := a.Store.PutRange(ctx, prefix, body); err != nil {
		log.Printf("cache write: %v", err)
	}
	return body, "MISS", nil
}

func (a *App) handleStats(w http.ResponseWriter, r *http.Request) {
	n, err := a.Store.TotalChecks(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "stats unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]int64{"totalChecks": n})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.Store.Ping(ctx); err != nil {
		httpError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func clientIP(r *http.Request) string {
	// Railway and most proxies put the real client first in X-Forwarded-For.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves built frontend files, falling back to index.html.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean("/" + r.URL.Path)
		if strings.HasPrefix(clean, "/api/") {
			httpError(w, http.StatusNotFound, "not found")
			return
		}
		if fi, err := os.Stat(filepath.Join(dir, clean)); err != nil || fi.IsDir() {
			if clean != "/" {
				r.URL.Path = "/"
			}
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(clean, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs.ServeHTTP(w, r)
	})
}
