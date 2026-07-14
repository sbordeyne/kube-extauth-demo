package openapi

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/sbordeyne/sample-api/pkg/db"
)

// ListUsers returns a page of users.
// (GET /v1/users)
func (s *Server) ListUsers(ctx context.Context, request ListUsersRequestObject) (ListUsersResponseObject, error) {
	limit, offset, page, perPage := paginate(request.Params.Page, request.Params.PerPage)

	rows, err := s.q.ListUsers(ctx, db.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	total, err := s.q.CountUsers(ctx)
	if err != nil {
		return nil, err
	}

	data := make([]User, len(rows))
	for i, row := range rows {
		data[i] = toAPIUser(row.User, row.UserRole)
	}
	return ListUsers200JSONResponse{Data: data, Meta: meta(total, page, perPage)}, nil
}

// CreateUser creates a user.
// (POST /v1/users)
func (s *Server) CreateUser(ctx context.Context, request CreateUserRequestObject) (CreateUserResponseObject, error) {
	body := request.Body
	if body == nil || body.Password == nil || *body.Password == "" {
		return CreateUser422JSONResponse{unprocessable("password is required")}, nil
	}

	role, err := s.q.GetRoleBySlug(ctx, string(body.Role))
	if err != nil {
		if isNotFound(err) {
			return CreateUser422JSONResponse{unprocessable("unknown role")}, nil
		}
		return nil, err
	}
	salt := generateSalt()
	hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password+*salt), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	created, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        string(body.Email),
		PasswordHash: string(hash),
		PasswordSalt: *salt,
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		AvatarUrl:    pgText(body.AvatarUrl),
		Bio:          pgText(body.Bio),
		RoleID:       role.ID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return CreateUser409JSONResponse{conflict("email already exists")}, nil
		}
		return nil, err
	}
	return CreateUser201JSONResponse(toAPIUser(created, role)), nil
}

// GetUser returns a user by id.
// (GET /v1/users/{userId})
func (s *Server) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	row, err := s.q.GetUser(ctx, pgUUID(request.UserId))
	if err != nil {
		if isNotFound(err) {
			return GetUser404JSONResponse{notFound("user not found")}, nil
		}
		return nil, err
	}
	return GetUser200JSONResponse(toAPIUser(row.User, row.UserRole)), nil
}

// UpdateUser partially updates a user.
// (PATCH /v1/users/{userId})
func (s *Server) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	body := request.Body
	if body == nil {
		return UpdateUser400JSONResponse{badRequest("empty body")}, nil
	}

	params := db.UpdateUserParams{
		ID:        pgUUID(request.UserId),
		FirstName: pgText(body.FirstName),
		LastName:  pgText(body.LastName),
		AvatarUrl: pgText(body.AvatarUrl),
		Bio:       pgText(body.Bio),
	}
	if body.Email != nil {
		email := string(*body.Email)
		params.Email = pgText(&email)
	}
	if body.Password != nil && *body.Password != "" {
		salt := generateSalt()
		hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password+*salt), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hs := string(hash)
		params.PasswordHash = pgText(&hs)
		params.PasswordSalt = pgText(salt)
	}
	if body.Role != nil {
		role, err := s.q.GetRoleBySlug(ctx, string(*body.Role))
		if err != nil {
			if isNotFound(err) {
				return UpdateUser422JSONResponse{unprocessable("unknown role")}, nil
			}
			return nil, err
		}
		params.RoleID = role.ID
	}

	updated, err := s.q.UpdateUser(ctx, params)
	if err != nil {
		if isNotFound(err) {
			return UpdateUser404JSONResponse{notFound("user not found")}, nil
		}
		if isUniqueViolation(err) {
			return UpdateUser409JSONResponse{conflict("email already exists")}, nil
		}
		return nil, err
	}
	return UpdateUser200JSONResponse(s.userWithRole(ctx, updated)), nil
}

// ReplaceUser fully replaces a user.
// (PUT /v1/users/{userId})
func (s *Server) ReplaceUser(ctx context.Context, request ReplaceUserRequestObject) (ReplaceUserResponseObject, error) {
	body := request.Body
	if body == nil || body.Password == nil || *body.Password == "" {
		return ReplaceUser422JSONResponse{unprocessable("password is required")}, nil
	}

	role, err := s.q.GetRoleBySlug(ctx, string(body.Role))
	if err != nil {
		if isNotFound(err) {
			return ReplaceUser422JSONResponse{unprocessable("unknown role")}, nil
		}
		return nil, err
	}
	salt := generateSalt()
	hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password+*salt), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	updated, err := s.q.ReplaceUser(ctx, db.ReplaceUserParams{
		ID:           pgUUID(request.UserId),
		Email:        string(body.Email),
		PasswordHash: string(hash),
		PasswordSalt: *salt,
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		AvatarUrl:    pgText(body.AvatarUrl),
		Bio:          pgText(body.Bio),
		RoleID:       role.ID,
	})
	if err != nil {
		if isNotFound(err) {
			return ReplaceUser404JSONResponse{notFound("user not found")}, nil
		}
		if isUniqueViolation(err) {
			return ReplaceUser409JSONResponse{conflict("email already exists")}, nil
		}
		return nil, err
	}
	return ReplaceUser200JSONResponse(toAPIUser(updated, role)), nil
}

// DeleteUser deletes a user.
// (DELETE /v1/users/{userId})
func (s *Server) DeleteUser(ctx context.Context, request DeleteUserRequestObject) (DeleteUserResponseObject, error) {
	if err := s.q.DeleteUser(ctx, pgUUID(request.UserId)); err != nil {
		return nil, err
	}
	return DeleteUser204Response{}, nil
}

// userWithRole loads the role for a bare user row so it can be returned to the API.
func (s *Server) userWithRole(ctx context.Context, u db.User) User {
	role, err := s.q.GetRole(ctx, u.RoleID)
	if err != nil {
		return toAPIUser(u, db.UserRole{ID: u.RoleID})
	}
	return toAPIUser(u, role)
}
