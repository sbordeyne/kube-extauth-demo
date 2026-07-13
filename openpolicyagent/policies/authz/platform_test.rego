package envoy.authz_test

import data.envoy.authz

jwt_signing_key := {
	"kty": "RSA",
	"n": "ofgWCuLjybRlzo0tZWJjNiuSfb4p4fAkd_wWJcyQoTbji9k0l8W26mPddxHmfHQp-Vaw-4qPCJrcS2mJPMEzP1Pt0Bm4d4QlL-yRT-SFd2lZS-pCgNMsD1W_YpRPEwOWvG6b32690r2jZ47soMZo9wGzjb_7OMg0LOL-bSf63kpaSHSXndS5z5rexMdbBYUsLA9e-KXBdQOS-UTo7WTBEMa2R2CapHg665xsmtdVMTBQY4uDZlxvb3qCo5ZwKh9kG4LT6_I5IhlJH7aGhyxXFvUK-DWNmoudF8NAco9_h9iaGNj8q2ethFkMLs91kzk2PAcDTW9gb54h4FRWyuXpoQ",
	"e": "AQAB",
	"d": "Eq5xpGnNCivDflJsRQBXHx1hdR1k6Ulwe2JZD50LpXyWPEAeP88vLNO97IjlA7_GQ5sLKMgvfTeXZx9SE-7YwVol2NXOoAJe46sui395IW_GO-pWJ1O0BkTGoVEn2bKVRUCgu-GjBVaYLU6f3l9kJfFNS3E0QbVdxzubSu3Mkqzjkn439X0M_V51gfpRLI9JYanrC4D4qAdGcopV_0ZHHzQlBjudU2QvXt4ehNYTCBr6XCLQUShb1juUO1ZdiYoFaFQT5Tw8bGUl_x_jTj3ccPDVZFD9pIuhLhBOneufuBiB4cS98l2SR_RQyGWSeWjnczT0QU91p1DhOVRuOopznQ",
	"p": "4BzEEOtIpmVdVEZNCqS7baC4crd0pqnRH_5IB3jw3bcxGn6QLvnEtfdUdiYrqBdss1l58BQ3KhooKeQTa9AB0Hw_Py5PJdTJNPY8cQn7ouZ2KKDcmnPGBY5t7yLc1QlQ5xHdwW1VhvKn-nXqhJTBgIPgtldC-KDV5z-y2XDwGUc",
	"q": "uQPEfgmVtjL0Uyyx88GZFF1fOunH3-7cepKmtH4pxhtCoHqpWmT8YAmZxaewHgHAjLYsp1ZSe7zFYHj7C6ul7TjeLQeZD_YwD66t62wDmpe_HlB-TnBA-njbglfIsRLtXlnDzQkv5dTltRJ11BKBBypeeF6689rjcJIDEz9RWdc",
	"dp": "BwKfV3Akq5_MFZDFZCnW-wzl-CCo83WoZvnLQwCTeDv8uzluRSnm71I3QCLdhrqE2e9YkxvuxdBfpT_PI7Yz-FOKnu1R6HsJeDCjn12Sk3vmAktV2zb34MCdy7cpdTh_YVr7tss2u6vneTwrA86rZtu5Mbr1C1XsmvkxHQAdYo0",
	"dq": "h_96-mK1R_7glhsum81dZxjTnYynPbZpHziZjeeHcXYsXaaMwkOlODsWa7I9xXDoRwbKgB719rrmI2oKr6N3Do9U0ajaHF-NKJnwgjMd2w9cjz3_-kyNlxAr2v4IKhGNpmM5iIgOS1VZnOZ68m6_pbLBSp3nssTdlqvd0tIiTHU",
	"qi": "IYd7DHOhrWvxkwPQsRM2tOgrjbcrfvtQJipd-DlcxyVuuM9sQLdgjVk2oy26F0EmpScGLq2MowX7fhd_QJQ3ydy5cY7YIBi87w93IKLEdfnbJtoOPLUW0ITrJReOgo1cq9SbsxYawBgfp_gh6A5603k2-ZQwVK0JKSHuLFkuQ3U",
}

jwt_headers := {
	"alg": "RS256",
	"typ": "JWT",
	"kid": "SvbozgHjJ6AlHg4m9RPhMxC7BIOv3eI3Qqu3b1wWQOA",
}

make_jwt_payload(exp, roles) := payload if {
	payload := {
		"exp": exp,
		"iat": exp,
		"auth_time": exp,
		"jti": "onrtac:354be421-3e9c-46b3-9d40-94d33d102e7f",
		"iss": "http://auth.polygone.localhost/realms/org-acme",
		"aud": "account",
		"sub": "265ae18f-f118-43c4-983a-0c29788a8717",
		"typ": "Bearer",
		"azp": "platform-frontend",
		"sid": "68f31cb5-e4d4-4e62-8d47-96f20f51a23d",
		"acr": "1",
		"allowed-origins": [
			"http://app.polygone.localhost",
			"http://localhost:5173",
		],
		"realm_access": {"roles": array.concat(
			[
				"default-roles-org-acme",
				"offline_access",
				"uma_authorization",
			],
			roles,
		)},
		"resource_access": {"account": {"roles": [
			"manage-account",
			"manage-account-links",
			"view-profile",
		]}},
		"scope": "openid profile email",
		"realm_roles": array.concat(
			[
				"default-roles-org-acme",
				"offline_access",
				"uma_authorization",
			],
			roles,
		),
		"email_verified": false,
		"name": "Admin User",
		"preferred_username": "admin@localhost",
		"given_name": "Admin",
		"family_name": "User",
		"email": "admin@localhost",
	}
}

now_plus_5m := floor((time.now_ns() + ((5 * 60) * 1e9)) / 1e9)

now_minus_5m := floor((time.now_ns() - ((5 * 60) * 1e9)) / 1e9)

org_admin_jwt := io.jwt.encode_sign(
	jwt_headers,
	make_jwt_payload(now_plus_5m, ["org:admin"]),
	jwt_signing_key,
)

platform_admin_jwt := io.jwt.encode_sign(
	jwt_headers,
	make_jwt_payload(now_plus_5m, ["platform:admin"]),
	jwt_signing_key,
)

platform_viewer_jwt := io.jwt.encode_sign(
	jwt_headers,
	make_jwt_payload(now_plus_5m, ["platform:viewer"]),
	jwt_signing_key,
)

expired_org_admin_jwt := io.jwt.encode_sign(
	jwt_headers,
	make_jwt_payload(now_minus_5m, ["org:admin"]),
	jwt_signing_key,
)

org_viewer_jwt := io.jwt.encode_sign(
	jwt_headers,
	make_jwt_payload(now_plus_5m, ["org:viewer"]),
	jwt_signing_key,
)

# Build envoy ext_authz request envelope.
envoy_input(method, path, token) := {"attributes": {"request": {"http": {
	"method": method,
	"path": path,
	"headers": auth_headers(token),
}}}}

auth_headers(token) := {"authorization": concat("", ["Bearer ", token])} if {
	token != ""
}

auth_headers(token) := {} if {
	token == ""
}

test_allow_cors_preflight if {
	authz.allow with input as envoy_input("OPTIONS", "/api/v1/orgs/register", "")
}

test_allow_org_registration if {
	authz.allow with input as envoy_input("POST", "/api/v1/orgs/register", "")
}

test_allow_public_catalog if {
	authz.allow with input as envoy_input("GET", "/api/v1/catalog", "")
}

test_allow_public_schemas if {
	authz.allow with input as envoy_input("GET", "/api/v1/schemas/downloader", "")
}

test_allow_org_admin_access_to_config_api if {
	authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/config/pipelines/1",
		org_admin_jwt,
	)
}

test_deny_org_admin_cross_org if {
	not authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/other/config/pipelines/1",
		org_admin_jwt,
	)
}

test_allow_platform_admin_access_to_any_org_api if {
	authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/config/pipelines/1",
		platform_admin_jwt,
	)
	authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/other/config/pipelines/1",
		platform_admin_jwt,
	)
}

test_allow_platform_viewer_read_only_access if {
	authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/config/pipelines/1",
		platform_viewer_jwt,
	)
	not authz.allow with input as envoy_input(
		"POST",
		"/api/v1/orgs/acme/config/pipelines/1",
		platform_viewer_jwt,
	)
}

test_allow_org_viewer_list_users if {
	authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/users/",
		org_viewer_jwt,
	)
}

test_deny_org_viewer_write if {
	not authz.allow with input as envoy_input(
		"POST",
		"/api/v1/orgs/acme/users/",
		org_viewer_jwt,
	)
}

test_deny_expired_token if {
	not authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/config/pipelines/1",
		expired_org_admin_jwt,
	)
}

test_deny_missing_token if {
	not authz.allow with input as envoy_input(
		"GET",
		"/api/v1/orgs/acme/config/pipelines/1",
		"",
	)
}
