package openapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/sbordeyne/sample-api/pkg/db"
)

// ---- pagination -------------------------------------------------------------

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// paginate normalizes page/per_page query params into SQL limit/offset and the
// effective values (for the response meta).
func paginate(page *int32, perPage *int32) (limit, offset, effPage, effPerPage int32) {
	effPage = int32(1)
	if page != nil && *page > 0 {
		effPage = *page
	}
	effPerPage = int32(defaultPerPage)
	if perPage != nil && *perPage > 0 {
		effPerPage = *perPage
	}
	if effPerPage > maxPerPage {
		effPerPage = maxPerPage
	}
	return effPerPage, (effPage - 1) * effPerPage, effPage, effPerPage
}

func meta(total int64, page, perPage int32) PaginationMeta {
	return PaginationMeta{Total: total, Page: page, PerPage: perPage}
}

// ---- pgtype <-> api helpers -------------------------------------------------

func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func toUUID(p pgtype.UUID) openapi_types.UUID {
	return uuid.UUID(p.Bytes)
}

func toUUIDPtr(p pgtype.UUID) *openapi_types.UUID {
	u := toUUID(p)
	return &u
}

func pgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

// ---- model mappers ----------------------------------------------------------

func toAPIRole(r db.UserRole) *Role {
	scopes := r.Scopes
	return &Role{
		Id:     toUUIDPtr(r.ID),
		Slug:   RoleSlug(r.Slug),
		Scopes: &scopes,
	}
}

func toAPIUser(u db.User, role db.UserRole) User {
	return User{
		Id:        toUUIDPtr(u.ID),
		Email:     openapi_types.Email(u.Email),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		AvatarUrl: textPtr(u.AvatarUrl),
		Bio:       textPtr(u.Bio),
		Role:      toAPIRole(role),
		CreatedAt: timePtr(u.CreatedAt),
		UpdatedAt: timePtr(u.UpdatedAt),
	}
}

func toAPIPost(p db.Post, commentCount int64) Post {
	cc := commentCount
	return Post{
		Id:           toUUIDPtr(p.ID),
		AuthorId:     toUUID(p.AuthorID),
		Title:        p.Title,
		Slug:         &p.Slug,
		Content:      p.Content,
		Excerpt:      textPtr(p.Excerpt),
		Status:       PostStatus(p.Status),
		Categories:   p.Categories,
		Tags:         p.Tags,
		CommentCount: &cc,
		CreatedAt:    timePtr(p.CreatedAt),
		UpdatedAt:    timePtr(p.UpdatedAt),
	}
}

func toAPIComment(c db.Comment) Comment {
	return Comment{
		Id:        toUUIDPtr(c.ID),
		PostId:    toUUIDPtr(c.PostID),
		AuthorId:  toUUID(c.AuthorID),
		Content:   c.Content,
		CreatedAt: timePtr(c.CreatedAt),
		UpdatedAt: timePtr(c.UpdatedAt),
	}
}

// ---- identity ---------------------------------------------------------------

type ctxKey string

const subKey ctxKey = "sub"

// identityFromCtx returns the caller UUID extracted from the (trusted) JWT sub
// claim by the identity middleware.
func identityFromCtx(ctx context.Context) (uuid.UUID, bool) {
	u, ok := ctx.Value(subKey).(uuid.UUID)
	return u, ok
}

// ---- error helpers ----------------------------------------------------------

func apiError(code int, msg string) Error {
	return Error{Code: int32(code), Error: msg}
}

// isNotFound reports whether err is a pgx "no rows" error.
func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// notFoundBody / etc. are the shared error bodies reused across handlers.
func notFound(msg string) NotFoundJSONResponse   { return NotFoundJSONResponse(apiError(http.StatusNotFound, msg)) }
func badRequest(msg string) BadRequestJSONResponse {
	return BadRequestJSONResponse(apiError(http.StatusBadRequest, msg))
}
func unauthorized(msg string) UnauthorizedJSONResponse {
	return UnauthorizedJSONResponse(apiError(http.StatusUnauthorized, msg))
}
func conflict(msg string) ConflictJSONResponse {
	return ConflictJSONResponse(apiError(http.StatusConflict, msg))
}
func unprocessable(msg string) UnprocessableEntityJSONResponse {
	return UnprocessableEntityJSONResponse(apiError(http.StatusUnprocessableEntity, msg))
}

func generateSalt() *string {
	salt, err := bcrypt.GenerateFromPassword([]byte(time.Now().String()), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	s := string(salt)
	return &s
}
