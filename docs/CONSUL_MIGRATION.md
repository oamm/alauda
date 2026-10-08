# Consul Migration

Consul registration:

```text
consul services register -name=Authentication.Grpc -address=lynx-authentication.lynx -port=81
```

Alauda registration:

```text
alauda environments create --key stg
alauda services create --name Authentication.Grpc
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81
```

Bootstrap is explicit and performed once, using an unrestricted registry.write credential. Registration requires registry.write; discovery clients need only discovery.read, optionally restricted to Environment. Re-running registration preserves identity and omitted fields/endpoints.

Discovery becomes `alauda services resolve Authentication.Grpc --environment stg --output value` or `GET /api/v1/discovery/Authentication.Grpc/resolve?environment=stg`. Output is `http://lynx-authentication.lynx:81/`. No Deployment knowledge or UUID lookup is required. Alauda does not implement the Consul wire protocol or load balancing; this is a command and API migration guide. The canonical rules are in [PUBLIC_API.md](PUBLIC_API.md), [CLI.md](CLI.md), and [SERVICE_DISCOVERY.md](SERVICE_DISCOVERY.md).
