package envoy.authz

import rego.v1

default allow := false

# ── Authorization rules ───────────────────────────────────────────────────────

allow if {
	http_request.method == "OPTIONS"
}
