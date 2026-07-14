# testing/ — authz scenarios

Black-box authorization tests that hit the live API at `api.demo.localhost` and
assert per-role outcomes. Each scenario is a JSON file; `run.sh` executes them with
`curl` + `jq`.

## Run

```sh
./run.sh                                # every testing/*.json
./run.sh user_admin_can_edit_posts.json # a single scenario
./run.sh 'user_admin_*.json'            # a glob (shell-expanded)

BASE_URL=http://api.demo.localhost ./run.sh   # override target (this is the default)
```

Prereqs: the stack must be up (`mise start`), and `curl` + `jq` on PATH (`jq` is
pinned in `mise.toml`). Exit code is non-zero if any scenario fails.

## Scenario format

A scenario is an ordered list of `steps`. Each step makes one HTTP request, checks
the status code (and optional `jq` conditions on the response body), and may
`capture` values into scenario-scoped variables. Later steps reference variables
with `{{var}}` in `path`, `body`, and `auth`.

```json
{
  "name": "user_admin_can_edit_self_comments",
  "description": "admin creates a comment then edits it",
  "steps": [
    { "request": { "method": "POST", "path": "/v1/login",
        "body": { "email": "admin@example.com", "password": "admin" } },
      "expect": { "status": 200 },
      "capture": { "token": ".token" } },
    { "auth": "{{token}}",
      "request": { "method": "POST", "path": "/v1/posts",
        "body": { "title": "t", "content": "c" } },
      "expect": { "status": 201 }, "capture": { "post_id": ".id" } },
    { "auth": "{{token}}",
      "request": { "method": "POST", "path": "/v1/posts/{{post_id}}/comments",
        "body": { "content": "hi" } },
      "expect": { "status": 201 }, "capture": { "cid": ".id" } },
    { "auth": "{{token}}",
      "request": { "method": "PUT", "path": "/v1/posts/{{post_id}}/comments/{{cid}}",
        "body": { "content": "edited" } },
      "expect": { "status": 200, "jq": [".content == \"edited\""] } }
  ]
}
```

Step fields:

| field            | required | meaning                                                                                                                       |
| ---------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `request.method` | yes      | HTTP method                                                                                                                   |
| `request.path`   | yes      | path appended to `BASE_URL`; supports `{{var}}`                                                                               |
| `request.body`   | no       | JSON object sent as the body; supports `{{var}}`                                                                              |
| `auth`           | no       | bearer token: `{{token}}`, or a crafted-JWT sentinel (`@wrongiss`/`@expired`/`@wrongkey`); omit for no `Authorization` header |
| `expect.status`  | yes      | exact status code the step must return                                                                                        |
| `expect.jq`      | no       | array of jq boolean expressions; all must be truthy against the response body                                                 |
| `capture`        | no       | map of `var` → jq expression evaluated against the response body                                                              |

Multi-user scenarios log in twice (`token`, `token2`) so one user creates a
resource and another acts on it — e.g. edit *others'* comments.

## Roles and expected outcomes

Seeded users (`<slug>@example.com` / `<slug>`): `admin`, `editor`, `reader`, `norole`.

- **admin** — full CRUD on posts/comments/users; edits **any** comment.
- **editor** — create posts/comments; edit/delete only **own** (`:me`).
- **reader** — read posts, create comments, edit only **own** comment.
- **norole** (`norole@example.com` / `norole`) — valid credentials but a role with an
  **empty scopes array**. Can log in; every scope-gated call must be denied.

## Crafted-JWT scenarios (`jwt_*`)

Token-validity checks, independent of role. `run.sh` mints these RS256 tokens at
startup and a step selects one with a sentinel `auth` value:

| sentinel    | token                                            | must be |
| ----------- | ------------------------------------------------ | ------- |
| `@wrongiss` | validly signed, unexpired, `iss != "sample-api"` | denied  |
| `@expired`  | validly signed, `exp` in the past                | denied  |
| `@wrongkey` | correct claims, signed with a throwaway key      | denied  |

Each token carries full admin scopes, so **only** the crafted flaw can cause the
denial. Every authenticated endpoint hit in these scenarios expects `403`.

Requires `openssl`. The signing key is read from
`JWT_KEY` (default `../sample-api/manifests/config/rs256.key`, populated by the
Tiltfile). If the key or `openssl` is missing, `run.sh` prints a note and those
scenarios send no token.

## Throwaway fixtures — negatives never touch seed data

While the policy is permissive, a negative test's forbidden action actually
**executes** (it returns 2xx instead of 403). So negatives must never act on the
seeded `admin`/`editor`/`reader` users — a `DELETE /v1/users/{admin_id}` that "should"
be 403 would really delete admin and break every later login.

Rule: any scenario that edits or deletes a user first **creates a throwaway user**
(as admin) and targets that id (`{{nid}}` / `{{target_uid}}`). Same idea for posts
and comments — every mutated post/comment is created within the scenario, never
assumed to pre-exist. The `{{nonce}}` variable (unique per scenario per run, seeded
by `run.sh`) keeps created user emails collision-free across repeated runs.

Read-only negatives (list/get) may reference seed users safely.

## Note on current state

The OPA policy (`openpolicyagent/policies/envoy/authz/platform.rego`) is currently an
allow-all stub. So positive scenarios pass, but the negative `*_cannot_*` scenarios
**fail** — that failure is the signal that scopes are not yet enforced. Once the
policy is tightened, every scenario should go green.

If a seeded user ever gets clobbered (e.g. by an older buggy scenario), restore it
with the exact seed values, e.g. for admin:

```sh
kubectl -n postgres exec postgres-0 -- psql -U demo -d demo -c \
  "INSERT INTO users (email, password_hash, password_salt, first_name, last_name, role_id) VALUES \
   ('admin@example.com', crypt('admin' || 'admin-seed-salt', gen_salt('bf')), 'admin-seed-salt', 'Admin', 'User', '00000000-0000-0000-0000-000000000001');"
```
