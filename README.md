# Kube-ExtAuth-Demo

Demo / workshop project to demonstrate how to use **kgateway** and **OpenPolicyAgent** to perform
authorization at the gateway level using Envoy Proxy's External Auth filter.

## Requirements

Install [`mise`](https://mise.jdx.dev/) and activate it.

```zsh
brew install mise
source <(mise activate)
mise install # install all project dependencies
```

You will also need `docker`, `curl` and `openssl` in your system `PATH`.
Make sure the Docker daemon is running before running the project

## Setup

Run the `start` task

```zsh
mise start
```

This will start `k3d` and will deploy the manifests locally using `tilt`. You can then visit the [Tilt Web UI](http://localhost:10350).

When you're done, you can use the `mise stop` command to tear down the cluster and delete persistent data.

## Usage

Edit the `openpolicyagent/policies/authz/platform.rego` file to test out rego policies, you can then test your changes with `mise test`

Check the provided [OpenAPI Spec file](./sample-api/openapi/schema.yaml) for more details.

You can generate a signed JWT (with the RS256 alg) using the `POST /v1/login` endpoint.

The public key is automatically mounted in `/config/rs256.pem` in **OpenPolicyAgent**'s pod.

## Exercises

The stup in `openpolicyagent/policies/authz/platform.rego` is currently allowing all requests.
The goal is to have all of the tests in the `testing/` suite pass, meaning that you'll have to:

- Validate JWT signing. A JWT signed with the wrong key should be denied
- Validate the JWT expiration, you should deny expired JWTs
- Validate the endpoint against the `scopes` claim in the JWT.

Scopes are defined as `<resource>:<action>(:<actor>)?  (actor "me" = own resources only)`.

Resources are:

- `post`
- `users`
- `comment`

Actions are:

- `list` : list resource endpoint (i.e. `GET /v1/users`)
- `read` : get a single resource endpoint (i.e. `GET /v1/users/:id`)
- `create` : create a resource (i.e. `POST /v1/users`)
- `edit` : Edit a resource (i.e. `PUT /v1/users/:id`, `PATCH /v1/users/:id`)
- `delete` : Delete a resource (i.e. `DELETE /v1/users/:id`)

**To go further**: you can add tests to the suite using the provided (AI-generated) `scripts/generate-scenarios.py`.
You could try to add authorization checks for specific fields when updating a user for instance.
