package envoy.authz

import rego.v1

cert := opa.runtime().env.RS256_CERT

default allow := false

allow if {
	input.attributes.request.http.method == "OPTIONS"
}

allow if {
	input.attributes.request.http.method == "GET"
}

allow if {
	input.attributes.request.http.method == "POST"
}

allow if {
	input.attributes.request.http.method == "PUT"
}

allow if {
	input.attributes.request.http.method == "DELETE"
}

allow if {
	input.attributes.request.http.method == "PATCH"
}
