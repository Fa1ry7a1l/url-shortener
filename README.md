# URL Shortener

A URL shortening service written in Go, with HTTP and gRPC APIs and a choice of PostgreSQL, file-backed or in-memory storage.

The service creates short links, redirects visitors to the original URL and lets users list and delete their links. HTTP clients are identified by a signed cookie, without a registration flow.

## Features

- Plain-text, JSON and batch HTTP endpoints; gRPC methods for shortening, resolving and listing links.
- PostgreSQL storage with automatic schema migrations, plus file and in-memory backends for local use.
- User-scoped link lists and asynchronous soft deletion through a bounded queue and batch worker.
- Gzip middleware, structured request logs and optional audit events to a file or HTTP receiver.
- Separate pprof listener, benchmarks, unit tests and graceful shutdown of the HTTP and gRPC servers.

## Quick start

Install Go 1.24.13 or later. Run all commands from the repository root.

```sh
git clone https://github.com/Fa1ry7a1l/url-shortener.git
cd url-shortener
go run ./cmd/shortener
```

This starts the HTTP API at `http://localhost:8080`, gRPC at `localhost:3200` and pprof at `http://localhost:6060/debug/pprof/`. The default storage is in memory; links are lost when the process stops.

In another terminal, create a short link and save the user cookie:

```sh
curl -i -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/articles/go"}' \
  http://localhost:8080/api/shorten
```

The response is `201 Created` with a JSON body containing `result`, the short URL. Open that URL in a browser to follow the redirect. To list links created with the same cookie:

```sh
curl -b cookies.txt http://localhost:8080/api/user/urls
```

See the [HTTP API reference](docs/http-api.md) for batch requests, deletion and response codes, or the [gRPC contract](api/shortener.proto) for RPC methods.

## Run with PostgreSQL

Docker Compose starts the database. The Go service runs separately:

```sh
docker compose up -d --wait postgres
go run ./cmd/shortener -d 'postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable'
```

The service applies SQL migrations on startup. Start it from the repository root so it can find `migrations/`. Database records persist in the Compose volume.

For a local file-backed setup instead:

```sh
go run ./cmd/shortener -f shortener-db.json
```

Settings can also come from a JSON file or environment variables. See [configuration](docs/configuration.md) and the [example config](examples/config.json).

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The tests use local storage and test servers; a running PostgreSQL instance is not required. [Development notes](docs/development.md) cover the custom analyzer, code generator, benchmarks and saved profiles.

## Project layout

| Path | Purpose |
| --- | --- |
| `cmd/shortener/` | Server entry point, dependency wiring and shutdown |
| `cmd/client/` | Interactive HTTP client for the default local endpoint |
| `cmd/staticlint/`, `cmd/reset/` | Static analysis and Reset method generation tools |
| `internal/handler/`, `internal/grpcserver/` | HTTP and gRPC transports |
| `internal/service/`, `internal/repository/` | URL operations and storage implementations |
| `internal/auth/`, `internal/audit/`, `internal/config/`, `internal/logger/`, `internal/buildinfo/` | Authentication, audit, settings, logging and build metadata |
| `api/`, `migrations/`, `pkg/pool/` | Protobuf contract, SQL migrations and a generic object pool |
| `docs/`, `examples/`, `profiles/` | Documentation, example settings and saved profiling data |

Both transports call the same service layer. Storage is selected at startup behind the `repository.Store` interface.

## Project background

Developed as part of the Yandex Practicum Go course, starting from the [URL shortener template](https://github.com/Yandex-Practicum/go-musthave-shortener-tpl). The repository includes the original course assessment workflows.
