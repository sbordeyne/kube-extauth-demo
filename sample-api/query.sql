-- Roles -----------------------------------------------------------------------

-- name: GetRole :one
SELECT * FROM user_roles WHERE id = $1;

-- name: GetRoleBySlug :one
SELECT * FROM user_roles WHERE slug = $1;

-- name: ListRoles :many
SELECT * FROM user_roles ORDER BY slug;

-- Users -----------------------------------------------------------------------
-- SELECTs embed the role so callers get scopes without a second query.

-- name: CreateUser :one
INSERT INTO users (
  email, password_hash, password_salt, first_name, last_name, avatar_url, bio, role_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetUser :one
SELECT sqlc.embed(users), sqlc.embed(user_roles)
FROM users
JOIN user_roles ON user_roles.id = users.role_id
WHERE users.id = $1;

-- name: GetUserByEmail :one
SELECT sqlc.embed(users), sqlc.embed(user_roles)
FROM users
JOIN user_roles ON user_roles.id = users.role_id
WHERE users.email = $1;

-- name: ListUsers :many
SELECT sqlc.embed(users), sqlc.embed(user_roles)
FROM users
JOIN user_roles ON user_roles.id = users.role_id
ORDER BY users.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: UpdateUser :one
-- Partial update: NULL args leave the column unchanged.
UPDATE users SET
  email         = COALESCE(sqlc.narg('email'), email),
  password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
  password_salt = COALESCE(sqlc.narg('password_salt'), password_salt),
  first_name    = COALESCE(sqlc.narg('first_name'), first_name),
  last_name     = COALESCE(sqlc.narg('last_name'), last_name),
  avatar_url    = COALESCE(sqlc.narg('avatar_url'), avatar_url),
  bio           = COALESCE(sqlc.narg('bio'), bio),
  role_id       = COALESCE(sqlc.narg('role_id'), role_id),
  updated_at    = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: ReplaceUser :one
UPDATE users SET
  email         = $2,
  password_hash = $3,
  password_salt = $4,
  first_name    = $5,
  last_name     = $6,
  avatar_url    = $7,
  bio           = $8,
  role_id       = $9,
  updated_at    = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- Posts -----------------------------------------------------------------------
-- comment_count is derived; dynamic API sort is not expressible in static SQL
-- so ordering is fixed here and the handler ignores the `sort` query param.

-- name: CreatePost :one
INSERT INTO posts (
  author_id, title, slug, content, excerpt, status, categories, tags
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetPost :one
SELECT
  sqlc.embed(posts),
  (SELECT count(*) FROM comments c WHERE c.post_id = posts.id) AS comment_count
FROM posts
WHERE posts.id = $1;

-- name: ListPosts :many
SELECT
  sqlc.embed(posts),
  (SELECT count(*) FROM comments c WHERE c.post_id = posts.id) AS comment_count
FROM posts
WHERE (sqlc.narg('status')::post_status IS NULL OR posts.status = sqlc.narg('status'))
  AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag') = ANY(posts.tags))
  AND (sqlc.narg('category')::text IS NULL OR sqlc.narg('category') = ANY(posts.categories))
ORDER BY posts.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPosts :one
SELECT count(*)
FROM posts
WHERE (sqlc.narg('status')::post_status IS NULL OR posts.status = sqlc.narg('status'))
  AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag') = ANY(posts.tags))
  AND (sqlc.narg('category')::text IS NULL OR sqlc.narg('category') = ANY(posts.categories));

-- name: ReplacePost :one
UPDATE posts SET
  title      = $2,
  slug       = $3,
  content    = $4,
  excerpt    = $5,
  status     = $6,
  categories = $7,
  tags       = $8,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- Comments --------------------------------------------------------------------

-- name: CreateComment :one
INSERT INTO comments (post_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetComment :one
SELECT * FROM comments WHERE id = $1;

-- name: ListComments :many
SELECT * FROM comments
WHERE post_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountComments :one
SELECT count(*) FROM comments WHERE post_id = $1;

-- name: UpdateComment :one
UPDATE comments SET
  content    = $2,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;
