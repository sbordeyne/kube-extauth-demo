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
    cmd = """
    mkdir -p jwt-test-server/config && \\
    openssl genrsa -out jwt-test-server/manifests/config/keypair.pem 2048 && \\
    openssl rsa -in jwt-test-server/manifests/config/keypair.pem -pubout -out jwt-test-server/manifests/config/rs256.pem && \\
    openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in jwt-test-server/manifests/config/keypair.pem -out jwt-test-server/manifests/config/rs256.key && \\
    cp -f jwt-test-server/manifests/config/rs256.pem openpolicyagent/config/rs256.pem
    """
    local_resource(
      "jwt-keypair",
      cmd=cmd,
      labels=["infra"],
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
    resource_deps=["kgateway", "jwt-keypair"],
    wait_cmd="kubectl wait deployment/open-policy-agent -n open-policy-agent --for=condition=Available --timeout=120s",
)

resource(
    "gateway",
    "gateway",
    resource_deps=["kgateway", "open-policy-agent"],
)

resource(
    "jwt-test-server",
    "jwt-test-server/manifests",
    resource_deps=["gateway", "jwt-keypair"],
    wait_cmd="kubectl wait deployment/jwt-test-server -n jwt-test-server --for=condition=Available --timeout=120s",
)

resource(
    "httpbin",
    "httpbin",
    resource_deps=["gateway"],
    wait_cmd="kubectl wait deployment/httpbin -n httpbin --for=condition=Available --timeout=120s"
)
