# HTTP API

[Back to README](../README.md)

Examples use the default base URL, `http://localhost:8080`.

## Endpoints

| Method | Path | Request | Success |
| --- | --- | --- | --- |
| `POST` | `/` | Original URL as `text/plain` | `201`, short URL as text |
| `POST` | `/api/shorten` | `{"url":"https://example.com"}` | `201`, `{"result":"http://localhost:8080/<id>"}` |
| `POST` | `/api/shorten/batch` | Array of `correlation_id` and `original_url` objects | `201`, array of `correlation_id` and `short_url` objects |
| `GET` | `/{id}` | Short ID in the path | `307`, original URL in `Location` |
| `GET` | `/api/user/urls` | User cookie | `200`, array of `short_url` and `original_url` objects; `204` if empty |
| `DELETE` | `/api/user/urls` | JSON array of short IDs and user cookie | `202`, queued for deletion |
| `GET` | `/ping` | None | `200` if the selected storage is available |
| `GET` | `/api/internal/stats` | Trusted `X-Real-IP` address | `200`, `{"urls":0,"users":0}` |

The HTTP middleware issues an `auth_token` cookie when no valid cookie is present. Keep the same cookie when creating, listing and deleting your links. An expired or invalid cookie creates a new anonymous user; it does not recover access to links owned by the old user.

## Create and list links

```sh
curl -i -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/articles/go"}' \
  http://localhost:8080/api/shorten

curl -i -b cookies.txt http://localhost:8080/api/user/urls
```

For the plain-text endpoint:

```sh
curl -i -b cookies.txt -c cookies.txt \
  -H 'Content-Type: text/plain' \
  --data-raw 'https://example.com/articles/storage' \
  http://localhost:8080/
```

Creating an original URL that already exists returns `409 Conflict` and the existing short URL for the two single-link endpoints. Deduplication is global, so creating a duplicate does not transfer its ownership to the current user.

## Batch creation

```sh
curl -i -b cookies.txt -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '[{"correlation_id":"first","original_url":"https://example.com/batch/1"},{"correlation_id":"second","original_url":"https://example.com/batch/2"}]' \
  http://localhost:8080/api/shorten/batch
```

Each response item retains its `correlation_id`. Empty batches, missing correlation IDs, invalid URLs and storage errors return `400 Bad Request`. Duplicate handling differs from the single-link endpoints; batch creation does not return existing URLs with `409`.

## Resolve and delete

Replace `<id>` with the final path segment of a returned short URL:

```sh
curl -i 'http://localhost:8080/<id>'

curl -i -b cookies.txt \
  -X DELETE \
  -H 'Content-Type: application/json' \
  -d '["<id>"]' \
  http://localhost:8080/api/user/urls
```

Resolution returns `404 Not Found` for an unknown ID and `410 Gone` for a deleted link. Deletion only affects IDs owned by the requesting user.

`202 Accepted` means the IDs were queued, not that storage has already changed. The worker groups IDs by user and flushes batches of up to 256 items or every 100 ms. The queue holds 4096 items. Deletion is not durable: queued work can be lost on a crash, and storage errors are not retried. A deletion request can also be partially queued before an invalid ID or a full queue causes a `400` response.

## Compression and diagnostics

Send `Content-Encoding: gzip` for compressed request bodies. With `Accept-Encoding: gzip`, JSON and HTML responses are compressed; `curl --compressed` handles this automatically.

`/ping` checks the active storage. The internal statistics endpoint uses the [trusted subnet setting](configuration.md#internal-statistics). Profiling routes are served on the separate pprof listener, not the main API port.

The router currently returns `400 Bad Request` for unmatched paths and unsupported methods.
