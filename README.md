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

## Setup

Run the `start` task

```zsh
mise start
```

This will start `k3d` and will deploy the manifests locally using `tilt`. You can then visit the [Tilt Web UI](http://localhost:10350).

## Usage

Edit the `openpolicyagent/policies/authz/platform.rego` file to test out rego policies, you can then test your changes with `curl`

```zsh
curl -sL -XOPTIONS http://api.demo.localhost/status/200
```

The exposed servers are [`httpbin`](https://httpbin.org), [under the go flavor](https://github.com/mccutchen/go-httpbin/tree/main),
and [`jsonplaceholder`](https://jsonplaceholder.typicode.com).

Check the provided [OpenAPI Spec file](./httpbin.openapi.yaml) for more details.

You can generate a signed JWT (with the RS256 alg) using the `curl -sL http://auth.demo.localhost/auth?ttl=5m` command.

The public key is automatically mounted in `/config/rs256.pem` in **OpenPolicyAgent**'s pod.

## Exercises

- Allow access to any `GET /status` API calls, but deny other HTTP methods
- Check that the JWT has the scope "html:read" in its `scopes` claim in order to access the `/html` endpoint
- Check that JWTs are signed with the proper signing key when provided for a protected endpoint.
- Check that the subject of the JWT matches the user id in the `GET /users/:id` endpoint. A regular user should be able to only see his/her data, not the others. Similarly the `GET /users/` list endpoint should be disallowed.
