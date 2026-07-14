package openapi

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/sbordeyne/sample-api/pkg/db"
)

// ListPosts returns a page of posts.
// (GET /v1/posts)
func (s *Server) ListPosts(ctx context.Context, request ListPostsRequestObject) (ListPostsResponseObject, error) {
	limit, offset, page, perPage := paginate(request.Params.Page, request.Params.PerPage)

	status := nullStatus(request.Params.Status)
	tag := pgText(request.Params.Tag)
	category := pgText(request.Params.Category)

	rows, err := s.q.ListPosts(ctx, db.ListPostsParams{
		Status:   status,
		Tag:      tag,
		Category: category,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	total, err := s.q.CountPosts(ctx, db.CountPostsParams{Status: status, Tag: tag, Category: category})
	if err != nil {
		return nil, err
	}

	data := make([]Post, len(rows))
	for i, row := range rows {
		data[i] = toAPIPost(row.Post, row.CommentCount)
	}
	return ListPosts200JSONResponse{Data: data, Meta: meta(total, page, perPage)}, nil
}

// CreatePost creates a post authored by the calling user.
// (POST /v1/posts)
func (s *Server) CreatePost(ctx context.Context, request CreatePostRequestObject) (CreatePostResponseObject, error) {
	author, ok := identityFromCtx(ctx)
	if !ok {
		return CreatePost401JSONResponse{unauthorized("authentication required")}, nil
	}
	body := request.Body
	if body == nil {
		return CreatePost400JSONResponse{badRequest("empty body")}, nil
	}

	created, err := s.q.CreatePost(ctx, db.CreatePostParams{
		AuthorID:   pgUUID(author),
		Title:      body.Title,
		Slug:       slugify(body.Title),
		Content:    body.Content,
		Excerpt:    pgText(body.Excerpt),
		Status:     postStatus(body.Status),
		Categories: deref(body.Categories),
		Tags:       deref(body.Tags),
	})
	if err != nil {
		return nil, err
	}
	return CreatePost201JSONResponse(toAPIPost(created, 0)), nil
}

// GetPost returns a post by id.
// (GET /v1/posts/{postId})
func (s *Server) GetPost(ctx context.Context, request GetPostRequestObject) (GetPostResponseObject, error) {
	row, err := s.q.GetPost(ctx, pgUUID(request.PostId))
	if err != nil {
		if isNotFound(err) {
			return GetPost404JSONResponse{notFound("post not found")}, nil
		}
		return nil, err
	}
	return GetPost200JSONResponse(toAPIPost(row.Post, row.CommentCount)), nil
}

// ReplacePost fully replaces a post.
// (PUT /v1/posts/{postId})
func (s *Server) ReplacePost(ctx context.Context, request ReplacePostRequestObject) (ReplacePostResponseObject, error) {
	body := request.Body
	if body == nil {
		return ReplacePost400JSONResponse{badRequest("empty body")}, nil
	}

	updated, err := s.q.ReplacePost(ctx, db.ReplacePostParams{
		ID:         pgUUID(request.PostId),
		Title:      body.Title,
		Slug:       slugify(body.Title),
		Content:    body.Content,
		Excerpt:    pgText(body.Excerpt),
		Status:     postStatus(body.Status),
		Categories: deref(body.Categories),
		Tags:       deref(body.Tags),
	})
	if err != nil {
		if isNotFound(err) {
			return ReplacePost404JSONResponse{notFound("post not found")}, nil
		}
		return nil, err
	}
	// comment_count not re-counted on replace; report 0 to avoid an extra query.
	return ReplacePost200JSONResponse(toAPIPost(updated, 0)), nil
}

// DeletePost deletes a post.
// (DELETE /v1/posts/{postId})
func (s *Server) DeletePost(ctx context.Context, request DeletePostRequestObject) (DeletePostResponseObject, error) {
	if err := s.q.DeletePost(ctx, pgUUID(request.PostId)); err != nil {
		return nil, err
	}
	return DeletePost204Response{}, nil
}

// ---- helpers ----------------------------------------------------------------

func nullStatus(s *PostStatus) db.NullPostStatus {
	if s == nil {
		return db.NullPostStatus{}
	}
	return db.NullPostStatus{PostStatus: db.PostStatus(*s), Valid: true}
}

func postStatus(s *PostStatus) db.PostStatus {
	if s == nil {
		return db.PostStatusDraft
	}
	return db.PostStatus(*s)
}

func deref(s *[]string) []string {
	if s == nil {
		return []string{}
	}
	return *s
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

// slugify builds a URL-safe slug from a title, suffixed with a short random id
// to guarantee uniqueness against the posts.slug unique constraint.
func slugify(title string) string {
	base := slugStrip.ReplaceAllString(strings.ToLower(title), "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "post"
	}
	return base + "-" + uuid.NewString()[:8]
}
