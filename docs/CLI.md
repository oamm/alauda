# CLI Reference

Global flags:

```bash
registryctl --server http://localhost:9700 --output pretty
registryctl --token "$REGISTRY_TOKEN" service list
```

`--output` supports `pretty`, `json`, and `yaml`. `REGISTRY_TOKEN` can be used instead of `--token`.

## Auth

```bash
registryctl auth login --username admin --password password --output json
registryctl auth me
registryctl auth user-create --username ops --email ops@example.local --display-name Ops --password password --role Operator
registryctl auth user-list
registryctl auth token-create --user-id <user-id> --name ci --scope read,write --expires-in 720h
registryctl auth token-list --user-id <user-id>
registryctl auth token-revoke <token-id>
```

## Audit Logs

```bash
registryctl audit-log --actor admin --page-size 50
registryctl audit-log --resource-type Catalog --resource-id <id> --output yaml
```

## Existing Domains

The CLI also supports environments, services, deployments, instances, endpoints, health checks, incidents, events, alert policies, and notification channels. Use `registryctl <command> --help` for command-specific flags.
