package openapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sbordeyne/sample-api/pkg/auth"
	"github.com/sbordeyne/sample-api/pkg/db"
)

// Server implements StrictServerInterface backed by Postgres.
type Server struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	signer *auth.Signer
}

var _ StrictServerInterface = (*Server)(nil)

// NewServer builds the API server.
func NewServer(pool *pgxpool.Pool, signer *auth.Signer) *Server {
	return &Server{pool: pool, q: db.New(pool), signer: signer}
}

// Handler returns the fully wired public HTTP handler.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(identity)

	strict := NewStrictHandler(s, nil)
	return HandlerFromMux(strict, r)
}

// identity extracts the caller UUID from the bearer JWT's sub claim and stores
// it in the request context. The token is parsed WITHOUT signature verification
// — it is assumed already validated/authorized by OPA at the gateway. Requests
// without a usable token simply carry no identity; handlers that require one
// respond 401.
func identity(next http.Handler) http.Handler {
	parser := jwt.NewParser()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw != "" {
			var claims jwt.RegisteredClaims
			if _, _, err := parser.ParseUnverified(raw, &claims); err == nil {
				if sub, err := uuid.Parse(claims.Subject); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), subKey, sub))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if rest, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(rest)
	}
	return ""
}
