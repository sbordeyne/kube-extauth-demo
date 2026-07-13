package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
  HTTP_HOST = ":8080"
  ADMIN_HOST = ":9090"

  PUBLIC_KEY_PATH = "/etc/jwt-test-server/rs256.pem"
  PRIVATE_KEY_PATH = "/etc/jwt-test-server/rs256.key"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
  w.Write([]byte("OK"))
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
  query := r.URL.Query()
  ttl, err := time.ParseDuration(query.Get("ttl"))
  if err != nil {
    http.Error(w, fmt.Sprintf("Invalid TTL: %v", err), http.StatusBadRequest)
    return
  }
  jti, err := uuid.NewV7()
  if err != nil {
    http.Error(w, fmt.Sprintf("Failed to generate JTI: %v", err), http.StatusInternalServerError)
    return
  }

  claims := jwt.MapClaims{
    "iss": "jwt-test-server",
    "sub": "test-user",
    "aud": "test-audience",
    "exp": time.Now().Add(ttl).Unix(),
    "iat": time.Now().Unix(),
    "jti": jti.String(),
  }
  for key, values := range query {
    if key != "ttl" {
      claims[key] = values[0]
    }
  }

  token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
  privateKeyData, err := os.ReadFile(PRIVATE_KEY_PATH)
  if err != nil {
    http.Error(w, fmt.Sprintf("Failed to read private key: %v", err), http.StatusInternalServerError)
    return
  }
  privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
  if err != nil {
    http.Error(w, fmt.Sprintf("Failed to parse private key: %v", err), http.StatusInternalServerError)
    return
  }
  signedToken, err := token.SignedString(privateKey)
  if err != nil {
    http.Error(w, fmt.Sprintf("Failed to sign token: %v", err), http.StatusInternalServerError)
    return
  }

  w.WriteHeader(http.StatusOK)
  w.Write([]byte(signedToken))
}

func main() {
  adminMux := http.NewServeMux()
  adminMux.HandleFunc("/health", HealthHandler)
  mainMux := http.NewServeMux()
  mainMux.HandleFunc("/auth", AuthHandler)

  adminSrv := &http.Server{Addr: ADMIN_HOST, Handler: adminMux}
  mainSrv := &http.Server{Addr: HTTP_HOST, Handler: mainMux}

  // Trigger shutdown of both servers on SIGTERM/SIGINT.
  ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
  defer stop()

  // Start both servers in separate goroutines. ErrServerClosed is the expected
  // result of a graceful Shutdown, so it is not treated as a failure. Any other
  // listen error triggers shutdown of the other server via stop().
  wg := &sync.WaitGroup{}
  wg.Add(2)
  go func() {
    defer wg.Done()
    fmt.Printf("Starting admin server on %s\n", ADMIN_HOST)
    if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
      fmt.Printf("Admin server failed: %v\n", err)
      stop()
    }
  }()
  go func() {
    defer wg.Done()
    fmt.Printf("Starting HTTP server on %s\n", HTTP_HOST)
    if err := mainSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
      fmt.Printf("HTTP server failed: %v\n", err)
      stop()
    }
  }()

  // Wait for a shutdown signal (or a server failure), then drain both servers
  // concurrently, bounded by a timeout.
  <-ctx.Done()
  fmt.Println("Shutdown signal received, draining...")
  shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
  for _, srv := range []*http.Server{adminSrv, mainSrv} {
    go func(s *http.Server) { s.Shutdown(shutdownCtx) }(srv)
  }

  wg.Wait()
}
