package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

// knownHints are the explicit type hints accepted as a "key:hint" suffix.
var knownHints = map[string]bool{"string": true, "int": true, "float": true, "bool": true, "json": true}

// parseClaim turns a raw query key and its values into a claim key and a typed
// value. A "key:hint" suffix forces the type of every value; otherwise each
// value is type-inferred. Repeated keys (multiple values) yield a []any array.
func parseClaim(rawKey string, values []string) (string, any, error) {
  key := rawKey
  hint := ""
  if i := strings.LastIndex(rawKey, ":"); i >= 0 && knownHints[rawKey[i+1:]] {
    key, hint = rawKey[:i], rawKey[i+1:]
  }

  if len(values) == 1 {
    v, err := coerceOrInfer(hint, values[0])
    return key, v, err
  }
  arr := make([]any, len(values))
  for i, s := range values {
    v, err := coerceOrInfer(hint, s)
    if err != nil {
      return key, nil, err
    }
    arr[i] = v
  }
  return key, arr, nil
}

// coerceOrInfer applies an explicit hint, or infers the scalar type when hint is empty.
func coerceOrInfer(hint, s string) (any, error) {
  switch hint {
  case "":
    return inferScalar(s), nil
  case "string":
    return s, nil
  case "int":
    return strconv.ParseInt(s, 10, 64)
  case "float":
    return strconv.ParseFloat(s, 64)
  case "bool":
    return strconv.ParseBool(s)
  case "json":
    var v any
    if err := json.Unmarshal([]byte(s), &v); err != nil {
      return nil, err
    }
    return v, nil
  default:
    return nil, fmt.Errorf("unknown type hint %q", hint)
  }
}

// inferScalar guesses a scalar type: bool (exact true/false), then int64, then
// float64, falling back to the raw string.
func inferScalar(s string) any {
  switch s {
  case "true":
    return true
  case "false":
    return false
  }
  if i, err := strconv.ParseInt(s, 10, 64); err == nil {
    return i
  }
  if f, err := strconv.ParseFloat(s, 64); err == nil {
    return f
  }
  return s
}

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

  // Default iss; the client may override it via a query param. exp/iat/jti are
  // forced by the server below so the client cannot override them.
  claims := jwt.MapClaims{"iss": "jwt-test-server"}
  for rawKey, values := range query {
    if rawKey == "ttl" {
      continue
    }
    key, val, err := parseClaim(rawKey, values)
    if err != nil {
      http.Error(w, fmt.Sprintf("Invalid claim %q: %v", rawKey, err), http.StatusBadRequest)
      return
    }
    claims[key] = val
  }
  now := time.Now()
  claims["exp"] = now.Add(ttl).Unix()
  claims["iat"] = now.Unix()
  claims["jti"] = jti.String()

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
