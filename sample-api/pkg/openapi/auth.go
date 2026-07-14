package openapi

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = time.Hour

// Login authenticates by email/password and returns a signed JWT.
// (POST /v1/login)
func (s *Server) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	if request.Body == nil || request.Body.Password == nil {
		return Login422JSONResponse{unprocessable("email and password are required")}, nil
	}

	row, err := s.q.GetUserByEmail(ctx, string(request.Body.Email))
	if err != nil {
		if isNotFound(err) {
			return Login401JSONResponse{unauthorized("invalid credentials")}, nil
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(row.User.PasswordHash), []byte(*request.Body.Password+row.User.PasswordSalt)); err != nil {
		return Login401JSONResponse{unauthorized("invalid credentials")}, nil
	}

	token, expiresAt, err := s.signer.Sign(toUUID(row.User.ID), row.UserRole.Slug, row.UserRole.Scopes, tokenTTL)
	if err != nil {
		return nil, err
	}

	return Login200JSONResponse{
		Token:     token,
		TokenType: Bearer,
		ExpiresAt: expiresAt,
		User:      toAPIUser(row.User, row.UserRole),
	}, nil
}
