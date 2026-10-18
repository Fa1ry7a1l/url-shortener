# Database migrations

PostgreSQL schema changes are managed with `golang-migrate`. When the PostgreSQL backend starts, it applies all pending `*.up.sql` files. Matching `*.down.sql` files describe the reverse changes; the service does not run them automatically.

| Version | Change |
| --- | --- |
| `000001` | Create `short_urls` |
| `000002` | Add a unique index on the original URL |
| `000003` | Add link ownership through `user_id` |
| `000004` | Add the `is_deleted` soft-deletion flag |

Migrations are loaded from `file://migrations`, relative to the working directory. Start the service from the repository root, or include this directory in the working directory when packaging the executable.

For database startup commands, see the [README](../README.md#run-with-postgresql).
