---
sidebar_position: 3
title: CLI & Makefile
---

# CLI & Makefile

## Binary flags

```bash
webhull [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-config` | string | `deploy/config.yaml` | Path to the operational config file (required). |
| `-pages` | string | — | Path to the pages/site structure file. When set, enables split config mode. |
| `-validate` | bool | `false` | Validate the config and exit without starting the server. Exit code 0 on success, 1 on error. |
| `-version` | bool | `false` | Print the binary version and exit. |

### Examples

```bash
# Start with monolithic config
webhull -config site.yaml

# Start with split config
webhull -config deploy/config.yaml -pages site/pages.yaml

# Validate config without starting
webhull -config site.yaml -validate
# config OK — site="My Site" languages=[en de] pages=8

# Print version
webhull -version
# webhull 1.2.3
```

### Setting the version at build time

The version string is set via `-ldflags` at build time:

```bash
go build \
  -ldflags="-X github.com/layer87-labs/webhull/cmd/webhull/cmd.Version=1.2.3" \
  -o build/bin/webhull \
  ./cmd/webhull
```

When not set, `Version` defaults to `"dev"`.

---

## Makefile targets

Run `make help` to list all targets.

### Development

| Target | Description |
|--------|-------------|
| `make generate` | Regenerate templ templates (`templ generate`). Run after editing any `.templ` file. |
| `make build` | `generate` → compile binary to `build/bin/webhull`. |
| `make run` | `build` → run with `deploy/config.yaml` + `examples/multi-page/pages.yaml`. |
| `make dev` | Hot-reload development server via [air](https://github.com/air-verse/air). |

### Quality gate

| Target | Description |
|--------|-------------|
| `make check` | **Full quality gate.** Runs `fmt-check → vet → lint → shadow → test → audit`. Required before every commit. |
| `make fmt` | Format all handwritten Go files with `gofumpt`. |
| `make fmt-check` | Fail if any file needs formatting (CI mode). |
| `make vet` | `go vet ./...` |
| `make lint` | `staticcheck -checks=all,-ST1000,-U1000 ./...` |
| `make shadow` | Variable shadow analysis (excludes generated `*_templ.go` files). |
| `make test` | `go test ./... -race -count=1 -v -buildvcs` |
| `make test-cover` | Tests with HTML coverage report → `build/coverage/coverage.html`. |
| `make audit` | `go mod verify` + `govulncheck` (binary mode when binary exists). |

### Maintenance

| Target | Description |
|--------|-------------|
| `make tidy` | `go mod tidy` + `go mod verify`. |
| `make clean` | Remove `build/` and `build/coverage/`. |
| `make compress` | Compress binary with UPX (requires `upx` installed). |
| `make container` | Build container image via `deploy/Containerfile`. |

### Makefile variables

Override on the command line to match your environment:

| Variable | Default | Description |
|----------|---------|-------------|
| `STATICCHECK_CHECKS` | `all,-ST1000,-U1000` | Staticcheck check set. Matches `.layer87/pipeline.yml`. |
| `TEST_FLAGS` | `-v -buildvcs` | Extra flags passed to `go test`. |
| `AUDIT_BINARY` | `build/bin/webhull` | Binary path for `govulncheck -mode binary`. |

```bash
# Override for CI without binary mode
make audit AUDIT_BINARY=""
```
