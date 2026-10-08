# Docker Image

The production image contains the registry server and the public CLI.

```bash
docker build -t alauda/service-registry:latest .
docker run --rm -p 9700:9700 -v alauda-data:/data alauda/service-registry:latest
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
