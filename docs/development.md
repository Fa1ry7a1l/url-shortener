# Development

[Back to README](../README.md)

Run commands from the repository root with Go 1.24.13 or later.

## Build and checks

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The race detector requires a supported platform and a C compiler. Tests use temporary files and local test servers; these commands do not exercise a live PostgreSQL database.

To build an executable:

```sh
mkdir -p bin
go build -o bin/shortener ./cmd/shortener
./bin/shortener
```

Keep the repository root as the working directory when running with PostgreSQL, because migration paths are relative to it.

## Tools

| Command | Purpose |
| --- | --- |
| `go run ./cmd/client` | Prompt for a URL and send it to `http://localhost:8080/` |
| `go run ./cmd/staticlint ./...` | Run the project's analyzer suite, including Staticcheck, bodyclose, nilerr and a custom check for direct `os.Exit` calls in `main` |
| `go run ./cmd/reset -root .` | Generate `reset.gen.go` files for structs marked with `// generate:reset` |

The static analyzer has its own [package documentation](../cmd/staticlint/doc.go). The reset generator changes source files; run it only when updating marked structs. Generated Protobuf sources are committed under `api/`, so a normal build does not require `protoc`.

## Benchmarks and profiling

The `BenchmarkShortenerUserURLs` benchmark measures listing 1024 links for one user in a store containing four users.

```sh
go test ./internal/service -run '^$' -bench '^BenchmarkShortenerUserURLs$' -benchmem
```

Capture a memory profile and inspect it:

```sh
mkdir -p .local
go test ./internal/service -run '^$' -bench '^BenchmarkShortenerUserURLs$' \
  -memprofile .local/current.pprof -o .local/service.test
go tool pprof -top -alloc_space .local/current.pprof
```

The repository also keeps historical before/after profiles. Compare them with:

```sh
go tool pprof -top -alloc_space -diff_base profiles/base.pprof profiles/result.pprof
```

These snapshots show allocation differences for the captured benchmark runs, not a throughput guarantee. Use the same Go version, hardware and benchmark settings for new comparisons.

The running service exposes pprof at `http://localhost:6060/debug/pprof/`. The address can be changed with `-pprof-address` or `PPROF_ADDRESS`.

## Course workflows

The existing GitHub Actions workflows are retained from the course. `shortenertest.yml` validates `iter<number>` pull-request branch names and runs the corresponding external assessment tests; pushes to `main` run all configured increments. `statictest.yml` runs the course's external vet tool. These workflows are separate from the local build and test commands above and may reject pull requests with other branch names.
