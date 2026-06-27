# Go Rewrite Architecture

## Runtime Shape

- `cmd/kkk-go-bot`: process entrypoint.
- `internal/app`: HTTP server, `/healthz`, Telegram webhook.
- `internal/config`: env/config loading.
- `internal/storage`: SQLite repository and migrations.
- `internal/legacy`: read-only import from PHP/JSON state.
- `internal/telegram`: Telegram Bot API client and update models.
- `internal/services`: compose service registry and availability probes.
- `internal/usecase`: admin gate, menu, routed bot use cases.

## DB Decision

Default: SQLite in `/data/bot.db`.

Reason: single-server bot, Docker volume backup is simple, no external DB dependency. PostgreSQL can be added later behind the same `storage.Repository` interface for multi-node/runtime HA.

## Current Migration Boundary

Imported now:

- admins from `app/config.php` when parsable
- redacted legacy config snapshot
- `config/pac.json`
- WireGuard client JSON files
- `config/hwid.json`
- service registry rows from compose

Not yet DB-owned:

- rendered WireGuard/Xray/AdGuard/OC/Naive/Hysteria config rendering
- full mutation parity for every legacy callback
- updater and backup/export v2

## Service Availability Rule

Menu entries are displayed only when both are true:

- service exists/enabled in compose
- service probe passes or no probe exists

Unavailable direct callbacks return an alert instead of mutating state.

## Cutover Rule

Keep PHP runtime default until Go handlers reach parity. Run Go with:

```sh
docker compose --profile go-bot up -d bot
```

Rollback: stop `bot`, keep PHP `php` and `service` containers.
