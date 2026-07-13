allow_k8s_contexts("k3d-polygone")
update_settings(max_parallel_updates=2)


def infra_resource(name, path, resource_deps=[], labels=["infra"], wait_cmd=""):
    cmd = "kustomize build --enable-helm {path} | kubectl apply --server-side --force-conflicts -f -".format(
        path=path
    )
    if wait_cmd:
        cmd += " && " + wait_cmd
    local_resource(
        "infra-" + name,
        cmd=cmd,
        deps=[path],
        resource_deps=resource_deps,
        labels=labels,
    )

infra_resource(
    "kgateway",
    "kgateway",
    wait_cmd="kubectl wait deployment -n kgateway-system --all --for=condition=available --timeout=120s",
)

infra_resource(
    "opa",
    "openpolicyagent",
    wait_cmd="kubectl wait deployment/open-policy-agent -n infra --for=condition=Available --timeout=120s",
)

infra_resource(
    "gateway",
    "gateway",
    resource_deps=["infra-kgateway", "infra-opa"],
)
