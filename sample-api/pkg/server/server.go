// Package server runs the public and health HTTP servers together and shuts
// them down gracefully.
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const shutdownTimeout = 15 * time.Second

// Run starts the public and health servers, each in its own goroutine, and
// blocks until ctx is cancelled (SIGINT/SIGTERM) or one server fails. Both
// goroutines report to a shared channel; on exit both servers are drained with
// a bounded timeout.
func Run(ctx context.Context, publicAddr, healthAddr string, public, health http.Handler) error {
	publicSrv := &http.Server{Addr: publicAddr, Handler: public}
	healthSrv := &http.Server{Addr: healthAddr, Handler: health}

	errCh := make(chan error, 2)
	serve := func(name string, srv *http.Server) {
		log.Printf("%s server listening on %s", name, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}
	go serve("public", publicSrv)
	go serve("health", healthSrv)

	var runErr error
	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case runErr = <-errCh:
		log.Printf("server error: %v", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := publicSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("public server shutdown: %v", err)
	}
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("health server shutdown: %v", err)
	}
	return runErr
}

// HealthHandler serves Kubernetes liveness and readiness probes.
func HealthHandler(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/healthz/ready", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return r
}
