# gRPC API

The [Protobuf contract](shortener.proto) defines `shortener.ShortenerService`:

| Method | Request | Response |
| --- | --- | --- |
| `ShortenURL` | `URLShortenRequest` with `url` | `URLShortenResponse` with `result`, the short URL |
| `ExpandURL` | `URLExpandRequest` with `id` | `URLExpandResponse` with `result`, the original URL |
| `ListUserURLs` | `google.protobuf.Empty` | `UserURLsResponse` containing `URLData` items |

The default address is `localhost:3200`. Server reflection is enabled. `shortener.pb.go` and `shortener_grpc.pb.go` are generated sources committed alongside the contract.

To use the HTTP user's identity, send the `auth_token` cookie value as `authorization` metadata, either as a raw token or with the `Bearer` prefix. `ListUserURLs` requires a valid user identity. Unlike the HTTP middleware, gRPC does not automatically issue an anonymous user token.

`ShortenURL` returns `AlreadyExists` for an existing original URL, with the short URL in the error message. `ExpandURL` returns `NotFound` for an unknown ID and `FailedPrecondition` for a deleted link.

See the [README](../README.md) for startup instructions and [configuration](../docs/configuration.md) for the address and TLS settings.
