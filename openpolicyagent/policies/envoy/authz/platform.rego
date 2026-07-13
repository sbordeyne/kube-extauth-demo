package envoy.authz

import rego.v1

default allow := false

allow if {
	input.attributes.request.http.method == "OPTIONS"
}
