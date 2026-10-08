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

## Build From Source

Requirements:

- Go 1.25 or newer
- Node.js 22 or newer
- npm
- Docker, when building or running the container image

Install dependencies and run the test suites:

```sh
make setup
make test
cd web && npm test -- --run
```

Build the server and CLI:

```sh
make build
make build-alauda
```

Build and run a local image:

```sh
make docker-build IMAGE=alauda:local
docker run --rm -p 9700:9700 -v "$(pwd)/data:/data" \
  -e ALAUDA_STORAGE_PROVIDER=sqlite \
  -e ALAUDA_DATABASE_URL=/data/alauda.db \
  alauda:local
```

Start the recommended PostgreSQL development workflow with `./scripts/dev.sh` or `make dev`. To use a local SQLite file instead, run `./scripts/dev.sh --db sqlite`. Run the frontend separately with `make dev-ui` when you need the Vite development server.

## Local Development Tutorial

Install dependencies once:

```sh
make setup
```

Start the full development app with PostgreSQL:

```sh
make dev
```

Or run the helper script directly:

```sh
./scripts/dev.sh
```

On Windows PowerShell, use the underlying script:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres
```

The script starts PostgreSQL with Docker Compose, applies migrations, builds the embedded UI, and serves Alauda at `http://127.0.0.1:9700`. On the first run it creates the `root` administrator and prints a temporary password. Sign in with that temporary password, then change it when prompted.

To use SQLite instead of PostgreSQL:

```sh
./scripts/dev.sh --db sqlite
```

Reset the local development initial password when you lose it or want to rotate it:

```sh
./scripts/dev.sh --reset-dev-password
```

On Windows PowerShell:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres -ResetDevPassword
```

The reset keeps the existing development database and records, revokes existing sessions, writes a new temporary password to the bootstrap credential file, and prints it in the terminal. After signing in, change the password again.

## Development With Docker Compose

The repository Compose file starts a PostgreSQL-backed development environment:

```sh
docker compose up --build
```

The web console is available at `http://127.0.0.1:9700`. Stop the stack with:

```sh
docker compose down
```

Add `-v` to `docker compose down` only when you intend to remove the local PostgreSQL and registry volumes.

## Operational Endpoints

- Web console and API: `http://127.0.0.1:9700`
- Liveness check: `http://127.0.0.1:9700/healthz`
- OpenAPI and API details: see [API overview](docs/API.md) and [Public API](docs/PUBLIC_API.md)

The `alauda` CLI talks to the server API and does not access the database directly. Set `ALAUDA_URL` and `ALAUDA_TOKEN` when using it in automation. See the [CLI guide](docs/CLI.md).

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

The web console is served by the registry binary. The default server port is `9700`; use `registry server --help` or the configuration documentation for available options.

## Contributing

Read the project documentation and run the Go and frontend test suites before opening a pull request. Keep API contracts and migration behavior documented when changing server or UI workflows.

## License

Alauda is licensed under the [Apache License 2.0](LICENSE).
