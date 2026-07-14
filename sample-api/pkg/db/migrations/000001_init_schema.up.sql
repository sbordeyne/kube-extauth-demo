-- DDL kept identical to schema.sql (canonical sqlc source); seed data appended.

CREATE EXTENSION IF NOT EXISTS pgcrypto;         -- gen_random_uuid()

CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE user_roles (
  id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug   text NOT NULL UNIQUE,
  scopes text[] NOT NULL DEFAULT '{}'
);

CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  password_salt text NOT NULL,
  first_name    text NOT NULL,
  last_name     text NOT NULL,
  avatar_url    text,
  bio           text,
  role_id       uuid NOT NULL REFERENCES user_roles (id),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_role_id ON users (role_id);

CREATE TABLE posts (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  author_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  title      text NOT NULL,
  slug       text NOT NULL UNIQUE,
  content    text NOT NULL,
  excerpt    text,
  status     post_status NOT NULL DEFAULT 'draft',
  categories text[] NOT NULL DEFAULT '{}',
  tags       text[] NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_posts_author_id ON posts (author_id);
CREATE INDEX idx_posts_status    ON posts (status);

CREATE TABLE comments (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id    uuid NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
  author_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  content    text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_comments_post_id   ON comments (post_id);
CREATE INDEX idx_comments_author_id ON comments (author_id);

-- Seed roles. Deterministic UUIDs so slug<->id mapping is stable across envs.
-- Scope form: <resource>:<action>(:<actor>)?  (actor "me" = own resources only)
INSERT INTO user_roles (id, slug, scopes) VALUES
  ('00000000-0000-0000-0000-000000000001', 'admin', ARRAY[
    'post:read', 'post:create', 'post:edit', 'post:delete', 'post:list',
    'comment:read', 'comment:create', 'comment:edit', 'comment:delete',
    'users:read', 'users:create', 'users:edit', 'users:delete'
  ]),
  ('00000000-0000-0000-0000-000000000002', 'editor', ARRAY[
    'post:read', 'post:create', 'post:list', 'post:edit:me', 'post:delete:me',
    'comment:read', 'comment:create', 'comment:edit:me', 'comment:delete:me',
    'users:read'
  ]),
  ('00000000-0000-0000-0000-000000000003', 'reader', ARRAY[
    'post:read', 'post:list',
    'comment:read', 'comment:create', 'comment:edit:me',
    'users:read:me'
  ]),
  ('00000000-0000-0000-0000-000000000004', 'norole', ARRAY[]::text[]);

-- Passwords are hashed as bcrypt(password || password_salt) to match the Go
-- login verification (auth.go). Credentials: <slug>@example.com / <slug>.
INSERT INTO users (email, password_hash, password_salt, first_name, last_name, role_id) VALUES
  ('admin@example.com',  crypt('admin'  || 'admin-seed-salt',  gen_salt('bf')), 'admin-seed-salt',  'Admin',  'User', '00000000-0000-0000-0000-000000000001'),
  ('editor@example.com', crypt('editor' || 'editor-seed-salt', gen_salt('bf')), 'editor-seed-salt', 'Editor', 'User', '00000000-0000-0000-0000-000000000002'),
  ('reader@example.com', crypt('reader' || 'reader-seed-salt', gen_salt('bf')), 'reader-seed-salt', 'Reader', 'User', '00000000-0000-0000-0000-000000000003'),
  ('norole@example.com', crypt('norole' || 'norole-seed-salt', gen_salt('bf')), 'norole-seed-salt', 'NoRole', 'User', '00000000-0000-0000-0000-000000000004');
