# Alauda Service Registry

Alauda is a self-hosted service registry and service discovery platform for registering services, environments, instances, endpoints, health checks, incidents, and execution results. It gives teams a clear operational view of distributed services without depending on a hosted control plane.

Run Alauda with the public container image:

```sh
docker pull oamm/alauda:latest
```

## Features

- Service, environment, instance, and endpoint registration
- Health checks, execution history, incidents, and recovery workflows
- REST and ConnectRPC APIs with OpenAPI documentation
- Web console for day-to-day operations
- `alauda` CLI for automation and public management workflows
- PostgreSQL and SQLite-backed persistence
- Runtime registration and service discovery workflows

## Choose a Storage Provider

Alauda supports SQLite for a simple single-node installation and PostgreSQL for shared or production deployments. Set both `ALAUDA_STORAGE_PROVIDER` and `ALAUDA_DATABASE_URL` explicitly in production.

### SQLite

SQLite is convenient for local development, evaluation, and a single Alauda instance:

```sh
docker volume create alauda-data
docker run --name alauda \
  -p 9700:9700 \
  -v alauda-data:/data \
  -e ALAUDA_STORAGE_PROVIDER=sqlite \
  -e ALAUDA_DATABASE_URL=/data/alauda.db \
  oamm/alauda:latest
```

The database is stored in the `alauda-data` volume. Back up that volume or the database file before upgrading or moving the installation.

### PostgreSQL

PostgreSQL is recommended when multiple users, replicas, or shared operational data are expected. The following example starts PostgreSQL and Alauda on the same Docker network:

```sh
docker network create alauda

docker run --name alauda-postgres --network alauda -d \
  -e POSTGRES_DB=alauda \
  -e POSTGRES_USER=alauda \
  -e POSTGRES_PASSWORD=change-me \
  postgres:16-alpine

docker run --name alauda --network alauda -p 9700:9700 -d \
  -e ALAUDA_STORAGE_PROVIDER=postgres \
  -e ALAUDA_DATABASE_URL='postgres://alauda:change-me@alauda-postgres:5432/alauda?sslmode=disable' \
  oamm/alauda:latest
```

Use a managed PostgreSQL service or a secret manager for real deployments. Do not keep the example password in production configuration.

## Command-Line Interface

The `alauda` CLI provides automation-friendly access to service registration, discovery, health checks, and results:

```sh
alauda services list
alauda services resolve Authentication.Grpc --environment production --output value
alauda health checks list --service Authentication.Grpc --environment production
```

Set `ALAUDA_URL` and `ALAUDA_TOKEN` for scripts and CI jobs. The CLI uses the server API and does not require direct database access. See the [CLI guide](docs/CLI.md).

## API and Documentation

- [API overview](docs/API.md)
- [Public API](docs/PUBLIC_API.md)
- [UI API reference](docs/UI_API.md)
- [CLI guide](docs/CLI.md)
- [Configuration](docs/CONFIGURATION.md)
- [Docker usage](docs/DOCKER.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Service discovery](docs/SERVICE_DISCOVERY.md)
- [Health model](docs/HEALTH.md)
- [Development guide](docs/DEVELOPMENT.md)

The web console is served by the registry binary. The default server port is `9700`; use `registry server --help` or the configuration documentation for available options.

## Contributing

Read the project documentation and run the Go and frontend test suites before opening a pull request. Keep API contracts and migration behavior documented when changing server or UI workflows.

## License

Alauda is licensed under the [Apache License 2.0](LICENSE).
