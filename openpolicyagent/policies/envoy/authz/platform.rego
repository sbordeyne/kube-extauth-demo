package envoy.authz

import rego.v1

cert := opa.runtime().env.RS256_CERT

# API service, reached directly (not via the gateway) for ownership lookups so
# these calls do NOT re-enter ext_authz.
api_base := "http://sample-api.sample-api.svc.cluster.local:8080"

default allow := false

# CORS preflight.
allow if input.attributes.request.http.method == "OPTIONS"

# Login is public.
allow if {
	input.attributes.request.http.method == "POST"
	glob.match("/*/login", ["/"], input.attributes.request.http.path)
}

# Everything else: valid token AND the endpoint's scope is satisfied.
allow if {
	is_token_valid
	authorized
}

# ---- token -----------------------------------------------------------------

token := {"valid": valid, "payload": payload} if {
	[_, encoded] := split(input.attributes.request.http.headers.authorization, " ")
	[valid, _, payload] := io.jwt.decode_verify(encoded, {"iss": "sample-api", "cert": cert})
}

is_token_valid if {
	token.valid
	now := time.now_ns() / 1000000000
	now < token.payload.exp
}

scopes := token.payload.scopes

sub := token.payload.sub

has_scope(s) if s in scopes

# ---- routing helpers --------------------------------------------------------

method := input.attributes.request.http.method

# Path split into segments, e.g. "/v1/posts/ID/comments/CID" -> [v1 posts ID comments CID]
segs := split(trim(input.attributes.request.http.path, "/"), "/")

users_collection if {
	count(segs) == 2
	segs == ["v1", "users"]
}

user_id := segs[2] if {
	count(segs) == 3
	segs[0] == "v1"
	segs[1] == "users"
}

posts_collection if {
	count(segs) == 2
	segs == ["v1", "posts"]
}

post_id := segs[2] if {
	count(segs) == 3
	segs[0] == "v1"
	segs[1] == "posts"
}

comments_collection_pid := segs[2] if {
	count(segs) == 4
	segs[0] == "v1"
	segs[1] == "posts"
	segs[3] == "comments"
}

comment_ref := {"pid": segs[2], "cid": segs[4]} if {
	count(segs) == 5
	segs[0] == "v1"
	segs[1] == "posts"
	segs[3] == "comments"
}

# ---- users ------------------------------------------------------------------

authorized if {
	method == "GET"
	users_collection
	has_scope("users:read")
}

authorized if {
	method == "GET"
	user_id
	has_scope("users:read")
}

authorized if {
	method == "GET"
	has_scope("users:read:me")
	user_id == sub
}

authorized if {
	method == "POST"
	users_collection
	has_scope("users:create")
}

authorized if {
	method in {"PUT", "PATCH"}
	user_id
	has_scope("users:edit")
}

authorized if {
	method == "DELETE"
	user_id
	has_scope("users:delete")
}

# ---- posts ------------------------------------------------------------------

authorized if {
	method == "GET"
	posts_collection
	has_scope("post:list")
}

authorized if {
	method == "POST"
	posts_collection
	has_scope("post:create")
}

authorized if {
	method == "GET"
	post_id
	has_scope("post:read")
}

authorized if {
	method == "PUT"
	post_id
	has_scope("post:edit")
}

authorized if {
	method == "PUT"
	has_scope("post:edit:me")
	post_owner(post_id) == sub
}

authorized if {
	method == "DELETE"
	post_id
	has_scope("post:delete")
}

authorized if {
	method == "DELETE"
	has_scope("post:delete:me")
	post_owner(post_id) == sub
}

# ---- comments ---------------------------------------------------------------

authorized if {
	method == "GET"
	comments_collection_pid
	has_scope("comment:read")
}

authorized if {
	method == "POST"
	comments_collection_pid
	has_scope("comment:create")
}

authorized if {
	method == "PUT"
	comment_ref
	has_scope("comment:edit")
}

authorized if {
	method == "PUT"
	has_scope("comment:edit:me")
	comment_owner(comment_ref.pid, comment_ref.cid) == sub
}

authorized if {
	method == "DELETE"
	comment_ref
	has_scope("comment:delete")
}

authorized if {
	method == "DELETE"
	has_scope("comment:delete:me")
	comment_owner(comment_ref.pid, comment_ref.cid) == sub
}

# ---- ownership lookups (DB is only source of truth for author_id) -----------

post_owner(id) := owner if {
	resp := http.send({
		"method": "GET",
		"url": sprintf("%s/v1/posts/%s", [api_base, id]),
		"raise_error": false,
	})
	resp.status_code == 200
	owner := resp.body.author_id
}

comment_owner(pid, cid) := owner if {
	resp := http.send({
		"method": "GET",
		"url": sprintf("%s/v1/posts/%s/comments?per_page=200", [api_base, pid]),
		"raise_error": false,
	})
	resp.status_code == 200
	some c in resp.body.data
	c.id == cid
	owner := c.author_id
}
