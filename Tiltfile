allow_k8s_contexts("k3d-polygone")
update_settings(max_parallel_updates=2)


def resource(name, path, resource_deps=[], labels=["infra"], wait_cmd=""):
    cmd = "kustomize build --enable-helm {path} | kubectl apply --server-side --force-conflicts -f -".format(
        path=path
    )
    if wait_cmd:
        cmd += " && " + wait_cmd
    local_resource(
        name,
        cmd=cmd,
        deps=[path],
        resource_deps=resource_deps,
        labels=labels,
    )

def generate_jwt_keypair():
    # Run at Tiltfile EVAL time (blocking) so the keypair files exist before any
    # kustomize()/k8s_yaml() call reads them. A local_resource runs in the async
    # update loop, which is too late for eval-time configMapGenerator reads.
    # Idempotent: regenerates only when the keypair is missing, keeping reloads fast.
    cmd = """
    set -euo pipefail
    if [ ! -f .dev-data/secrets/jwt/keypair.pem ]; then
      mkdir -p .dev-data/secrets/jwt
      openssl genrsa -out .dev-data/secrets/jwt/keypair.pem 2048
      openssl rsa -in .dev-data/secrets/jwt/keypair.pem -pubout -out .dev-data/secrets/jwt/rs256.pem
      openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in .dev-data/secrets/jwt/keypair.pem -out .dev-data/secrets/jwt/rs256.key
    fi
    mkdir -p openpolicyagent/config sample-api/manifests/config
    # Copy only when content differs, else mtime bump retriggers the file watcher
    # -> reconcile -> re-eval -> copy -> infinite reconciliation loop.
    sync() { cmp -s "$1" "$2" || cp -f "$1" "$2"; }
    sync .dev-data/secrets/jwt/rs256.pem openpolicyagent/config/rs256.pem
    sync .dev-data/secrets/jwt/rs256.key sample-api/manifests/config/rs256.key
    sync .dev-data/secrets/jwt/rs256.pem sample-api/manifests/config/rs256.pem
    """
    local(cmd, quiet=True)

def backend(name, resource_deps=[]):
    local_resource(
        name + "-build",
        cmd="""
            set -euo pipefail
            cd {name}
            CGO_ENABLED=0 GOOS=linux go build -o ../.dev-data/tilt/{name} github.com/sbordeyne/{name}
            kubectl rollout restart deployment/{name} -n {name} 2>/dev/null || true
        """.format(name=name),
        deps=[name],
        ignore=["**/*_test.go"],
        labels=["backend"],
    )
    k8s_yaml(kustomize("{}/manifests".format(name)))
    k8s_resource(
        name,
        new_name=name,
        resource_deps=[name + "-build"] + resource_deps,
        labels=["backend"],
    )

generate_jwt_keypair()

resource(
    "kgateway",
    "kgateway",
    wait_cmd="kubectl wait deployment -n kgateway-system --all --for=condition=available --timeout=120s",
)

resource(
    "open-policy-agent",
    "openpolicyagent",
    resource_deps=["kgateway"],
    wait_cmd="kubectl wait deployment/open-policy-agent -n open-policy-agent --for=condition=Available --timeout=120s",
)

resource(
    "gateway",
    "gateway",
    resource_deps=["kgateway", "open-policy-agent"],
)

resource(
    "postgresql",
    "postgresql",
    wait_cmd="kubectl wait statefulset/postgres -n postgres --for=jsonpath=.status.readyReplicas=1 --timeout=120s",
)

backend("sample-api", resource_deps=["postgresql", "gateway"])
