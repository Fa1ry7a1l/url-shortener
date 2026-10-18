# Configuration

[Back to README](../README.md)

Settings are applied in this order, from lowest to highest priority:

1. Built-in defaults.
2. JSON config file selected with `-c` or `-config`.
3. Explicit command-line flags.
4. Non-empty environment variables.

`CONFIG` selects the config file and overrides the path passed with `-c` or `-config`. An invalid config file fails startup even if a flag or environment variable overrides its settings.

```sh
go run ./cmd/shortener -c examples/config.json
```

The example uses file storage in `.local/shortener.json`. With `go run`, put application flags after `./cmd/shortener`.

## Settings

| JSON key | Flag | Environment variable | Default |
| --- | --- | --- | --- |
| `server_address` | `-a` | `SERVER_ADDRESS` | `localhost:8080` |
| `grpc_server_address` | `-g`, `-grpc-address`, `-grpc-server-address` | `GRPC_SERVER_ADDRESS`, `GRPC_ADDRESS` | `localhost:3200` |
| `pprof_address` | `-pprof-address` | `PPROF_ADDRESS` | `localhost:6060` |
| `base_url` | `-b` | `BASE_URL` | `http://localhost:8080` |
| `id_length` | `-l` | — | `8` |
| `file_storage_path` | `-f` | `FILE_STORAGE_PATH` | Empty |
| `database_dsn` | `-d` | `DATABASE_DSN` | Empty |
| `auth_secret` | `-auth-secret` | `AUTH_SECRET` | `url-shortener-secret` |
| `enable_https` | `-s` | `ENABLE_HTTPS` | `false` |
| `audit_file` | `-audit-file` | `AUDIT_FILE` | Empty |
| `audit_url` | `-audit-url` | `AUDIT_URL` | Empty |
| `trusted_subnet` | `-t` | `TRUSTED_SUBNET` | Empty |

If both gRPC environment variables are set, `GRPC_SERVER_ADDRESS` wins. Non-positive ID lengths fall back to `8`.

`base_url` controls the links returned to clients; it does not change the HTTP listen address. Trailing slashes are removed.

## Storage selection

The first matching option is used:

1. Non-empty `database_dsn`: PostgreSQL, with migrations applied at startup.
2. Non-empty `file_storage_path`: JSON file, loaded at startup and written on changes.
3. Otherwise: in-memory storage.

A failed PostgreSQL connection stops startup; it does not fall back to file or memory storage. SQL migrations are read from `file://migrations`, relative to the working directory.

## Local diagnostics and TLS

The pprof server uses its own listener. Its default address is local to the machine; keep it private when changing bind addresses.

`enable_https` enables TLS on HTTP and gRPC using certificates generated at startup. The certificates are self-signed and intended for local development. Set `base_url` to an `https://` URL separately when enabling this option. The pprof listener remains HTTP.

The built-in authentication secret is a development default. Use `AUTH_SECRET` for a private deployment-specific value. Changing it invalidates existing user cookies.

## Internal statistics

`GET /api/internal/stats` is disabled with `403 Forbidden` unless `trusted_subnet` is configured. The handler checks the address in `X-Real-IP` against that CIDR. For a deployment behind a proxy, the proxy must overwrite this header and direct access to the backend must be restricted.

## Audit

`audit_file` appends JSON-lines events. `audit_url` sends events as HTTP POST requests; both can be enabled together. Successful single-link creation and resolution produce events. Audit delivery runs in the request path, and delivery errors are not returned to the API client. Batch creation and deletion do not emit audit events.
