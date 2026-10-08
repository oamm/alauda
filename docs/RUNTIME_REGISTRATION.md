# Runtime Registration

The stable public contract is Service -> Environment -> Instance -> Endpoint -> Health Check. See [PUBLIC_API.md](PUBLIC_API.md) for the canonical REST schema, reconciliation, errors and lifecycle semantics; [CLI.md](CLI.md) documents the same model through flags and files.

Service and Environment are explicit bootstrap prerequisites. Registration does not create them on typo. The public facade resolves their exact case-sensitive keys and uses the existing repositories/transaction internally. Instance identity is Service + Environment + Instance name; Endpoint identity is Instance + Endpoint name.

```bash
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81
alauda services resolve Authentication.Grpc --environment stg --output value
```

Registration is an atomic idempotent UPSERT. Omitted fields/endpoints preserve state. Explicit replacement retires omitted endpoints; explicit Primary promotion demotes the old Primary before writing the new one. Primary is inferred only for a new singleton when omitted. Health Check configuration is a separate operation; registration never invents a Healthy result.

Legacy RegisterRuntime RPC remains create-only and ID-addressed for existing frontend/administrative consumers. Its Deployment relationships are internal implementation details, not required by the public facade. Legacy CreateDeployment/CreateInstance/CreateEndpoint RPCs remain available during migration, subject to the same backend Environment authorization. Their removal requires migrating known frontend and singular CLI consumers first.
