# Docker Image

The production image contains the registry server and the public CLI.

```bash
docker build -t alauda/service-registry:latest .
docker run --rm -p 9700:9700 -v alauda-data:/data \
  -e ALAUDA_STORAGE_PROVIDER=sqlite \
  -e ALAUDA_DATABASE_URL=/data/alauda.db \
  alauda/service-registry:latest
```

The standalone example selects SQLite for a single-node deployment. For PostgreSQL, set `ALAUDA_STORAGE_PROVIDER=postgres` and either `ALAUDA_DATABASE_URL` or the component environment variables shown below. The container does not start or provision PostgreSQL.

For a database that already exists outside the Alauda container, point the host at an address reachable from inside Docker. On Docker Desktop, `host.docker.internal` reaches the host machine. On Linux Docker Engine, add the host-gateway mapping:

```bash
docker run --name alauda -p 9700:9700 -d \
  --add-host=host.docker.internal:host-gateway \
  -e ALAUDA_STORAGE_PROVIDER=postgres \
  -e ALAUDA_POSTGRES_HOST=host.docker.internal \
  -e ALAUDA_POSTGRES_PORT=5432 \
  -e ALAUDA_POSTGRES_USER=elephas \
  -e 'ALAUDA_POSTGRES_PASSWORD=15Qx(8qo<%,2' \
  -e ALAUDA_POSTGRES_DATABASE=alauda \
  -e ALAUDA_POSTGRES_SSLMODE=disable \
  oamm/alauda:latest
```

The component variables avoid manual URL encoding for passwords. If you prefer a single URL, percent-encode special characters and use the same reachable host:

```bash
docker run --name alauda -p 9700:9700 -d \
  --add-host=host.docker.internal:host-gateway \
  -e ALAUDA_STORAGE_PROVIDER=postgres \
  -e 'ALAUDA_DATABASE_URL=postgres://elephas:15Qx%288qo%3C%25%2C2@host.docker.internal:5432/alauda?sslmode=disable' \
  oamm/alauda:latest
```

The container starts the server through `/usr/local/bin/registry server`. The canonical CLI is `/usr/local/bin/alauda`; `registryctl` remains a compatibility alias:

```bash
docker run --rm --entrypoint alauda alauda/service-registry:latest --help
docker run --rm --entrypoint registryctl alauda/service-registry:latest services --help
```

For a registry publish, choose the fully qualified image name explicitly:

```bash
docker build -t ghcr.io/example/alauda:0.1.0 .
docker push ghcr.io/example/alauda:0.1.0
```

The Make target accepts the same override: `make docker-build IMAGE=ghcr.io/example/alauda:0.1.0` and `make docker-push IMAGE=ghcr.io/example/alauda:0.1.0`.

## Standalone executable

Build the native CLI without running Docker:

```bash
go build -o bin/alauda ./cmd/registryctl
```

On Windows, use `go build -o bin/alauda.exe ./cmd/registryctl`. The CLI consumes the public server API; it does not require a local database. Supply ALAUDA_URL and ALAUDA_TOKEN through the environment. CLI configuration never stores bearer credentials.
