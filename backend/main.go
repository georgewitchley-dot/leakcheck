// Command pwcheck serves the Password Leak Checker: a static React frontend and a
// small API that proxies Have I Been Pwned "range" lookups (k-anonymity), caching
// responses in Postgres. The server only ever sees the first 5 hex characters of
// a SHA-1 hash, never a password or a full hash.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := ":" + envOr("PORT", "8080")
	staticDir := envOr("STATIC_DIR", "./public")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := openStore(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer store.Close()

	app := &App{
		Store:    store,
		Upstream: NewHIBPClient(envOr("HIBP_BASE_URL", "https://api.pwnedpasswords.com")),
		CacheTTL: 24 * time.Hour,
		Limiter:  NewRateLimiter(60, time.Minute),
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Routes(staticDir),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
