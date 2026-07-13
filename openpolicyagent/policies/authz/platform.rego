package envoy.authz

import rego.v1

default allow := false

# ── Authorization rules ───────────────────────────────────────────────────────

allow if {
	http_request.method == "OPTIONS"
}

allow if {
	http_request.method == "POST"
	http_request.path == "/api/v1/orgs/register"
}

allow if {
	http_request.method == "GET"
	startswith(http_request.path, "/api/v1/schemas")
}

allow if {
	startswith(http_request.path, "/api/v1/catalog")
}

# Platform admin: unrestricted access across all orgs.
allow if {
	token_valid
	"platform:admin" in realm_roles
}

# Platform viewer: read-only across all orgs.
allow if {
	token_valid
	"platform:viewer" in realm_roles
	http_request.method == "GET"
}

# Org owner: full access within their org.
allow if {
	token_valid
	org_match
	"org:owner" in realm_roles
}

# Org admin: full access within their org.
allow if {
	token_valid
	org_match
	"org:admin" in realm_roles
}

# Org editor: full access within their org.
allow if {
	token_valid
	org_match
	"org:editor" in realm_roles
}

# Org viewer: read-only within their org.
allow if {
	token_valid
	org_match
	"org:viewer" in realm_roles
	http_request.method == "GET"
}

# Self-service: any authenticated org member can update their own notification preferences.
allow if {
	token_valid
	org_match
	endswith(http_request.path, "/notification-preferences")
	user_id_from_path == token.payload.sub
}

# ── Helpers ───────────────────────────────────────────────────────────────────

# Envoy ext_authz wraps the HTTP request under input.attributes.request.http.
http_request := input.attributes.request.http

user_id_from_path := uid if {
	parts := split(trim_prefix(http_request.path, "/"), "/")
	parts[2] == "orgs"
	parts[4] == "users"
	uid := parts[5]
}

# Merge custom mapper claim (realm_roles) with standard Keycloak claim
# (realm_access.roles) so the policy works even when the mapper is absent.
realm_roles contains role if {
	some role in token.payload.realm_roles
}

realm_roles contains role if {
	some role in token.payload.realm_access.roles
}

token_valid if {
	token.payload.exp * 1e9 > time.now_ns()
}

# Extract bearer token from Authorization header. Header keys are lowercase
# in envoy's ext_authz input.
bearer_token := t if {
	auth := http_request.headers.authorization
	startswith(auth, "Bearer ")
	t := substring(auth, 7, -1)
}

token := {"payload": payload} if {
	[_, payload, _] := io.jwt.decode(bearer_token)
}

# Extract org slug from the verified token issuer.
# iss format: http://auth.polygone.localhost/realms/org-{slug}
org_slug_from_token := slug if {
	contains(token.payload.iss, "/realms/org-")
	slug := split(token.payload.iss, "/realms/org-")[1]
}

# Extract org slug from the request path.
# Path format: /api/v1/orgs/{slug}/...
org_slug_from_path := slug if {
	parts := split(trim_prefix(http_request.path, "/"), "/")
	parts[2] == "orgs"
	slug := parts[3]
}

# Token org and URL org must agree.
org_match if {
	org_slug_from_token != ""
	org_slug_from_token == org_slug_from_path
}
