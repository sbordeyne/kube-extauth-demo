package openapi

import (
	"context"

	"github.com/sbordeyne/sample-api/pkg/db"
)

// ListComments returns a page of comments on a post.
// (GET /v1/posts/{postId}/comments)
func (s *Server) ListComments(ctx context.Context, request ListCommentsRequestObject) (ListCommentsResponseObject, error) {
	// Ensure the post exists so we can distinguish 404 from an empty list.
	if _, err := s.q.GetPost(ctx, pgUUID(request.PostId)); err != nil {
		if isNotFound(err) {
			return ListComments404JSONResponse{notFound("post not found")}, nil
		}
		return nil, err
	}

	limit, offset, page, perPage := paginate(request.Params.Page, request.Params.PerPage)

	rows, err := s.q.ListComments(ctx, db.ListCommentsParams{
		PostID: pgUUID(request.PostId),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	total, err := s.q.CountComments(ctx, pgUUID(request.PostId))
	if err != nil {
		return nil, err
	}

	data := make([]Comment, len(rows))
	for i, c := range rows {
		data[i] = toAPIComment(c)
	}
	return ListComments200JSONResponse{Data: data, Meta: meta(total, page, perPage)}, nil
}

// CreateComment adds a comment authored by the calling user.
// (POST /v1/posts/{postId}/comments)
func (s *Server) CreateComment(ctx context.Context, request CreateCommentRequestObject) (CreateCommentResponseObject, error) {
	author, ok := identityFromCtx(ctx)
	if !ok {
		return CreateComment401JSONResponse{unauthorized("authentication required")}, nil
	}
	if request.Body == nil {
		return CreateComment400JSONResponse{badRequest("empty body")}, nil
	}

	created, err := s.q.CreateComment(ctx, db.CreateCommentParams{
		PostID:   pgUUID(request.PostId),
		AuthorID: pgUUID(author),
		Content:  request.Body.Content,
	})
	if err != nil {
		// FK violation on a missing post surfaces as a unique/constraint error.
		if isUniqueViolation(err) {
			return CreateComment404JSONResponse{notFound("post not found")}, nil
		}
		return nil, err
	}
	return CreateComment201JSONResponse(toAPIComment(created)), nil
}

// UpdateComment updates a comment.
// (PUT /v1/posts/{postId}/comments/{commentId})
func (s *Server) UpdateComment(ctx context.Context, request UpdateCommentRequestObject) (UpdateCommentResponseObject, error) {
	if request.Body == nil {
		return UpdateComment400JSONResponse{badRequest("empty body")}, nil
	}
	updated, err := s.q.UpdateComment(ctx, db.UpdateCommentParams{
		ID:      pgUUID(request.CommentId),
		Content: request.Body.Content,
	})
	if err != nil {
		if isNotFound(err) {
			return UpdateComment404JSONResponse{notFound("comment not found")}, nil
		}
		return nil, err
	}
	return UpdateComment200JSONResponse(toAPIComment(updated)), nil
}

// DeleteComment deletes a comment.
// (DELETE /v1/posts/{postId}/comments/{commentId})
func (s *Server) DeleteComment(ctx context.Context, request DeleteCommentRequestObject) (DeleteCommentResponseObject, error) {
	if err := s.q.DeleteComment(ctx, pgUUID(request.CommentId)); err != nil {
		return nil, err
	}
	return DeleteComment204Response{}, nil
}
