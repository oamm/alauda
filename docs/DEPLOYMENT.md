# Deployment Guide

## Production Startup

```powershell
$env:REGISTRY_CONFIG="config.production.yaml"
registry server
```

Use `server.devMode: false`. Production API resources are protected by default. On the first start, Alauda creates the `root` administrator and writes a one-time temporary credential to `auth.bootstrapCredentialPath` (default `./data/bootstrap-admin-credential`) with restrictive permissions. Retrieve that file once, sign in, and change the password. The credential is never regenerated or printed on later restarts.

## Health And Metrics

| Endpoint   | Purpose                 |
| ---------- | ----------------------- |
| `/healthz` | Process liveness        |
| `/readyz`  | Database readiness      |
| `/version` | Binary version metadata; authenticated |
| `/metrics` | Prometheus text metrics; authenticated |

## SQLite Backups

Run SQLite in WAL mode, which the server enables on startup. Back up the database with the SQLite online backup API where possible. If filesystem backups are used, copy the `.db`, `.db-wal`, and `.db-shm` files together while the server is stopped or quiesced.

Recommended schedule:

- Hourly backups for high-change environments.
- Daily retained backups for 30 days.
- Monthly retained backups for 12 months.
- Restore tests at least quarterly.

## Retention

Detailed health results are pruned on startup using `health.resultRetentionHours`. Audit logs are retained indefinitely. Events, incidents, and availability history are retained for operational history and should be archived externally before manual cleanup.

## Security Checklist

- Keep authentication enabled; production cannot disable it.
- Keep `server.devMode` disabled.
- Protect the bootstrap credential file and remove it after securely recording the temporary credential.
- Replace the temporary root password on first login.
- Set `rateLimit.enabled`.
- Store config secrets outside source control.
- Scrape `/metrics` and alert on `/readyz` failures.
