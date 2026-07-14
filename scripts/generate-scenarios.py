#!/usr/bin/env python3
"""Generate the testing/ authz scenario JSON files.

Writes every generated scenario into ../testing (relative to this script). Files
listed in `existing` (the hand-written scenarios) are skipped, never overwritten.
Run after editing the scenario matrix below:  python3 scripts/generate-scenarios.py
"""
import json
import os

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "testing")

CREDS = {
    "admin":  ("admin@example.com",  "admin"),
    "editor": ("editor@example.com", "editor"),
    "reader": ("reader@example.com", "reader"),
    "norole": ("norole@example.com", "norole"),
}

def login(role, tokvar="token", extra=None):
    e, p = CREDS[role]
    cap = {tokvar: ".token"}
    if extra:
        cap.update(extra)
    return {
        "request": {"method": "POST", "path": "/v1/login",
                    "body": {"email": e, "password": p}},
        "expect": {"status": 200},
        "capture": cap,
    }

def step(method, path, tok, status, body=None, jq=None, capture=None):
    return raw_step(method, path, "{{%s}}" % tok, status, body, jq, capture)

def raw_step(method, path, auth, status, body=None, jq=None, capture=None):
    # auth is used verbatim (a captured-token template or a @sentinel like @expired).
    s = {"request": {"method": method, "path": path}, "expect": {"status": status}}
    if auth is not None:
        s["auth"] = auth
    if body is not None:
        s["request"]["body"] = body
    if jq:
        s["expect"]["jq"] = jq
    if capture:
        s["capture"] = capture
    return s

POST_BODY = {"title": "Scenario post", "content": "body"}
COMMENT_BODY = {"content": "a comment"}
def user_body(role="reader"):
    return {"email": "newuser-{{nonce}}@example.com", "password": "password123",
            "first_name": "New", "last_name": "User", "role": role}

def create_post(tok, cap="post_id"):
    return step("POST", "/v1/posts", tok, 201, POST_BODY, capture={cap: ".id"})
def create_comment(tok, cap="comment_id"):
    return step("POST", "/v1/posts/{{post_id}}/comments", tok, 201, COMMENT_BODY,
                capture={cap: ".id"})
def create_user(tok, cap="target_uid"):
    # A throwaway fixture user so edit/delete negative tests never mutate seed users.
    return step("POST", "/v1/users", tok, 201, user_body("reader"), capture={cap: ".id"})

scenarios = {}
def add(name, desc, steps):
    scenarios[name] = {"name": name, "description": desc, "steps": steps}

# ----------------------------------------------------------------------------
# POSTS
# ----------------------------------------------------------------------------
for role in ("admin", "editor", "reader"):
    add(f"user_{role}_can_list_posts", f"{role} has post:list and can list posts",
        [login(role), step("GET", "/v1/posts", "token", 200, jq=[".data != null"])])
    # read a single post (everyone has post:read); admin creates it first
    add(f"user_{role}_can_read_post", f"{role} has post:read and can read a post",
        [login("admin"),
         create_post("token"),
         login(role, "token2"),
         step("GET", "/v1/posts/{{post_id}}", "token2", 200, jq=[".id != null"])])

# admin
add("user_admin_can_edit_others_posts",
    "admin has unscoped post:edit — editor creates a post, admin replaces it",
    [login("admin"), login("editor", "token2"),
     step("POST", "/v1/posts", "token2", 201, POST_BODY, capture={"post_id": ".id"}),
     step("PUT", "/v1/posts/{{post_id}}", "token", 200,
          {"title": "Admin edit", "content": "x"}, jq=[".title == \"Admin edit\""])])
add("user_admin_can_delete_posts",
    "admin has post:delete and can delete its own post",
    [login("admin"), create_post("token"),
     step("DELETE", "/v1/posts/{{post_id}}", "token", 204)])
add("user_admin_can_delete_others_posts",
    "admin has unscoped post:delete — editor creates a post, admin deletes it",
    [login("admin"), login("editor", "token2"),
     step("POST", "/v1/posts", "token2", 201, POST_BODY, capture={"post_id": ".id"}),
     step("DELETE", "/v1/posts/{{post_id}}", "token", 204)])

# editor
add("user_editor_can_edit_self_posts",
    "editor has post:edit:me and can replace its own post",
    [login("editor"), create_post("token"),
     step("PUT", "/v1/posts/{{post_id}}", "token", 200,
          {"title": "Editor edit", "content": "x"}, jq=[".title == \"Editor edit\""])])
add("user_editor_cannot_edit_others_posts",
    "editor has only post:edit:me — replacing admin's post must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("editor", "token2"),
     step("PUT", "/v1/posts/{{post_id}}", "token2", 403,
          {"title": "nope", "content": "x"})])
add("user_editor_can_delete_self_posts",
    "editor has post:delete:me and can delete its own post",
    [login("editor"), create_post("token"),
     step("DELETE", "/v1/posts/{{post_id}}", "token", 204)])
add("user_editor_cannot_delete_others_posts",
    "editor has only post:delete:me — deleting admin's post must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("editor", "token2"),
     step("DELETE", "/v1/posts/{{post_id}}", "token2", 403)])

# reader (no post:edit / post:delete at all)
add("user_reader_cannot_edit_posts",
    "reader lacks post:edit — replacing a post must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("reader", "token2"),
     step("PUT", "/v1/posts/{{post_id}}", "token2", 403,
          {"title": "nope", "content": "x"})])
add("user_reader_cannot_delete_posts",
    "reader lacks post:delete — deleting a post must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("reader", "token2"),
     step("DELETE", "/v1/posts/{{post_id}}", "token2", 403)])

# ----------------------------------------------------------------------------
# COMMENTS
# ----------------------------------------------------------------------------
for role in ("admin", "editor", "reader"):
    add(f"user_{role}_can_list_comments",
        f"{role} has comment:read and can list comments on a post",
        [login("admin"), create_post("token"), login(role, "token2"),
         step("GET", "/v1/posts/{{post_id}}/comments", "token2", 200,
              jq=[".data != null"])])

# editor create comment (reader/admin already covered by hand-written files)
add("user_editor_can_create_comments",
    "editor has comment:create and can comment on a post",
    [login("admin"), create_post("token"), login("editor", "token2"),
     step("POST", "/v1/posts/{{post_id}}/comments", "token2", 201, COMMENT_BODY,
          jq=[".id != null"])])

# editor edit/delete own + cannot others
add("user_editor_can_edit_self_comments",
    "editor has comment:edit:me and can edit its own comment",
    [login("admin"), create_post("token"), login("editor", "token2"),
     create_comment("token2"),
     step("PUT", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 200,
          {"content": "editor edited own"}, jq=[".content == \"editor edited own\""])])
add("user_editor_cannot_edit_others_comments",
    "editor has only comment:edit:me — editing admin's comment must be forbidden (expects 403)",
    [login("admin"), create_post("token"), create_comment("token"),
     login("editor", "token2"),
     step("PUT", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 403,
          {"content": "nope"})])
add("user_editor_can_delete_self_comments",
    "editor has comment:delete:me and can delete its own comment",
    [login("admin"), create_post("token"), login("editor", "token2"),
     create_comment("token2"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 204)])
add("user_editor_cannot_delete_others_comments",
    "editor has only comment:delete:me — deleting admin's comment must be forbidden (expects 403)",
    [login("admin"), create_post("token"), create_comment("token"),
     login("editor", "token2"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 403)])

# admin delete own + others
add("user_admin_can_delete_self_comments",
    "admin can delete its own comment",
    [login("admin"), create_post("token"), create_comment("token"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token", 204)])
add("user_admin_can_delete_others_comments",
    "admin has unscoped comment:delete — editor comments, admin deletes it",
    [login("admin"), create_post("token"), login("editor", "token2"),
     create_comment("token2"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token", 204)])

# reader cannot delete (no comment:delete scope at all)
add("user_reader_cannot_delete_self_comments",
    "reader lacks comment:delete — deleting even its own comment must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("reader", "token2"),
     create_comment("token2"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 403)])
add("user_reader_cannot_delete_others_comments",
    "reader lacks comment:delete — deleting admin's comment must be forbidden (expects 403)",
    [login("admin"), create_post("token"), create_comment("token"),
     login("reader", "token2"),
     step("DELETE", "/v1/posts/{{post_id}}/comments/{{comment_id}}", "token2", 403)])

# ----------------------------------------------------------------------------
# USERS
# ----------------------------------------------------------------------------
# admin: full CRUD
add("user_admin_can_list_users", "admin has users:read and can list users",
    [login("admin"), step("GET", "/v1/users", "token", 200, jq=[".data != null"])])
add("user_admin_can_read_user", "admin has users:read and can read a user",
    [login("admin", extra={"uid": ".user.id"}),
     step("GET", "/v1/users/{{uid}}", "token", 200, jq=[".id != null"])])
add("user_admin_can_create_users", "admin has users:create and can create a user",
    [login("admin"),
     step("POST", "/v1/users", "token", 201, user_body("reader"),
          jq=[".id != null", ".role.slug == \"reader\""])])
add("user_admin_can_edit_users", "admin has users:edit and can update a user",
    [login("admin"),
     step("POST", "/v1/users", "token", 201, user_body("reader"),
          capture={"nid": ".id"}),
     step("PATCH", "/v1/users/{{nid}}", "token", 200, {"first_name": "Edited"},
          jq=[".first_name == \"Edited\""])])
add("user_admin_can_delete_users", "admin has users:delete and can delete a user",
    [login("admin"),
     step("POST", "/v1/users", "token", 201, user_body("reader"),
          capture={"nid": ".id"}),
     step("DELETE", "/v1/users/{{nid}}", "token", 204)])

# editor: users:read (unscoped) only
add("user_editor_can_list_users", "editor has users:read and can list users",
    [login("editor"), step("GET", "/v1/users", "token", 200, jq=[".data != null"])])
add("user_editor_can_read_others_user",
    "editor has unscoped users:read and can read another user",
    [login("admin"), create_user("token"), login("editor", "token2"),
     step("GET", "/v1/users/{{target_uid}}", "token2", 200, jq=[".id != null"])])
add("user_editor_cannot_create_users",
    "editor lacks users:create — creating a user must be forbidden (expects 403)",
    [login("editor"), step("POST", "/v1/users", "token", 403, user_body("reader"))])
add("user_editor_cannot_edit_users",
    "editor lacks users:edit — updating a user must be forbidden (expects 403)",
    [login("admin"), create_user("token"), login("editor", "token2"),
     step("PATCH", "/v1/users/{{target_uid}}", "token2", 403, {"first_name": "nope"})])
add("user_editor_cannot_delete_users",
    "editor lacks users:delete — deleting a user must be forbidden (expects 403)",
    [login("admin"), create_user("token"), login("editor", "token2"),
     step("DELETE", "/v1/users/{{target_uid}}", "token2", 403)])

# reader: users:read:me only
add("user_reader_can_read_self_user",
    "reader has users:read:me and can read its own user record",
    [login("reader", extra={"uid": ".user.id"}),
     step("GET", "/v1/users/{{uid}}", "token", 200,
          jq=[".email == \"reader@example.com\""])])
add("user_reader_cannot_read_others_user",
    "reader has only users:read:me — reading another user must be forbidden (expects 403)",
    [login("admin"), create_user("token"), login("reader", "token2"),
     step("GET", "/v1/users/{{target_uid}}", "token2", 403)])
add("user_reader_cannot_list_users",
    "reader lacks unscoped users:read — listing users must be forbidden (expects 403)",
    [login("reader"), step("GET", "/v1/users", "token", 403)])
add("user_reader_cannot_create_users",
    "reader lacks users:create — creating a user must be forbidden (expects 403)",
    [login("reader"), step("POST", "/v1/users", "token", 403, user_body("reader"))])
add("user_reader_cannot_edit_users",
    "reader lacks users:edit — updating a user must be forbidden (expects 403)",
    [login("admin"), create_user("token"), login("reader", "token2"),
     step("PATCH", "/v1/users/{{target_uid}}", "token2", 403, {"first_name": "nope"})])
add("user_reader_cannot_delete_users",
    "reader lacks users:delete — deleting a user must be forbidden (expects 403)",
    [login("admin"), create_user("token"), login("reader", "token2"),
     step("DELETE", "/v1/users/{{target_uid}}", "token2", 403)])

# ----------------------------------------------------------------------------
# NOROLE — valid credentials but a role with an empty scopes array. Login works;
# every scope-gated action must be denied.
# ----------------------------------------------------------------------------
add("user_norole_can_login",
    "norole logs in and receives a token whose role has no scopes",
    [{"request": {"method": "POST", "path": "/v1/login",
                  "body": {"email": "norole@example.com", "password": "norole"}},
      "expect": {"status": 200,
                 "jq": [".token != null", ".user.role.slug == \"norole\"",
                        "(.user.role.scopes | length) == 0"]}}])

add("user_norole_cannot_list_posts",
    "norole has no scopes — listing posts must be forbidden (expects 403)",
    [login("norole"), step("GET", "/v1/posts", "token", 403)])
add("user_norole_cannot_read_post",
    "norole has no scopes — reading a post must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("norole", "token2"),
     step("GET", "/v1/posts/{{post_id}}", "token2", 403)])
add("user_norole_cannot_create_posts",
    "norole has no scopes — creating a post must be forbidden (expects 403)",
    [login("norole"), step("POST", "/v1/posts", "token", 403, POST_BODY)])
add("user_norole_cannot_list_comments",
    "norole has no scopes — listing comments must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("norole", "token2"),
     step("GET", "/v1/posts/{{post_id}}/comments", "token2", 403)])
add("user_norole_cannot_create_comments",
    "norole has no scopes — commenting must be forbidden (expects 403)",
    [login("admin"), create_post("token"), login("norole", "token2"),
     step("POST", "/v1/posts/{{post_id}}/comments", "token2", 403, COMMENT_BODY)])
add("user_norole_cannot_list_users",
    "norole has no scopes — listing users must be forbidden (expects 403)",
    [login("norole"), step("GET", "/v1/users", "token", 403)])
add("user_norole_cannot_read_self_user",
    "norole has no scopes (not even users:read:me) — reading own user must be forbidden (expects 403)",
    [login("norole", extra={"uid": ".user.id"}),
     step("GET", "/v1/users/{{uid}}", "token", 403)])

# ----------------------------------------------------------------------------
# CRAFTED JWTs — token-validity checks, independent of role. Each token carries
# full admin scopes, so ONLY the crafted flaw can cause the denial. run.sh mints
# these from the @sentinel; every authenticated endpoint hit must return 403.
# ----------------------------------------------------------------------------
ZERO = "00000000-0000-0000-0000-000000000000"
def denied_calls(sentinel):
    return [
        raw_step("GET",    "/v1/posts",                       sentinel, 403),
        raw_step("POST",   "/v1/posts",                       sentinel, 403, POST_BODY),
        raw_step("GET",    f"/v1/posts/{ZERO}",               sentinel, 403),
        raw_step("GET",    f"/v1/posts/{ZERO}/comments",      sentinel, 403),
        raw_step("POST",   f"/v1/posts/{ZERO}/comments",      sentinel, 403, COMMENT_BODY),
        raw_step("GET",    "/v1/users",                       sentinel, 403),
        raw_step("POST",   "/v1/users",                       sentinel, 403, user_body("reader")),
    ]

add("jwt_wrong_issuer_is_denied",
    "a validly-signed, unexpired JWT with iss != 'sample-api' must be denied on every authenticated call (expects 403)",
    denied_calls("@wrongiss"))
add("jwt_expired_is_denied",
    "a validly-signed JWT with exp in the past must be denied on every authenticated call (expects 403)",
    denied_calls("@expired"))
add("jwt_wrong_signing_key_is_denied",
    "a JWT signed with the wrong key must be denied on every authenticated call (expects 403)",
    denied_calls("@wrongkey"))

# ----------------------------------------------------------------------------
existing = {
    "user_admin_can_login", "user_editor_can_login", "user_reader_can_login",
    "user_admin_can_create_posts", "user_admin_can_edit_posts",
    "user_editor_can_create_posts", "user_reader_cannot_create_posts",
    "user_admin_can_create_comments", "user_admin_can_edit_self_comments",
    "user_admin_can_edit_others_comments", "user_reader_can_create_comments",
    "user_reader_can_edit_self_comments", "user_reader_cannot_edit_others_comments",
}

written = 0
skipped = []
for name, sc in scenarios.items():
    if name in existing:
        skipped.append(name)
        continue
    with open(os.path.join(OUT, name + ".json"), "w") as f:
        json.dump(sc, f, indent=2)
        f.write("\n")
    written += 1

print(f"written: {written}")
print(f"skipped (already exist): {len(skipped)} -> {sorted(skipped)}")
