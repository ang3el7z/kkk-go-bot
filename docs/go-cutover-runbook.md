# Go Cutover Runbook

## Preconditions

- Linux server with Docker and existing PHP runtime healthy.
- `.env` contains `TELEGRAM_BOT_TOKEN`.
- Telegram long polling is enabled by default. Set `TELEGRAM_POLLING=0` only when an HTTPS webhook is configured externally.
- `app/config.php` exists for legacy import, but Go does not use the token from it.
- `docker compose --profile go-bot config --services` succeeds.
- `make go` starts `bot` and `curl http://127.0.0.1:8082/healthz` returns `{"status":"ok"}`.

## Backup

```sh
make backup
make go
```

In Telegram, run:

```text
/backup
```

Store both `backup.json` and `kkk-go-bot-backup.json`.

## Cutover

```sh
make go-cutover
```

Then validate:

```sh
docker compose ps bot php service
docker compose logs --tail=100 bot
curl http://127.0.0.1:8082/healthz
```

Telegram checks:

- `/id`
- `/menu`
- WireGuard menu opens only if `wg`/`wg1` containers are available.
- Xray menu opens only if `xr` is available.
- `/backup` returns JSON document.

## Rollback

```sh
make go-rollback
```

Rollback only stops the Go `bot` container. Existing PHP/service containers remain untouched.

## Post-Cutover Notes

- Keep PHP containers until one production cycle is validated.
- Do not delete `app/config.php`; importer still reads non-secret legacy state.
- Runtime validation requiring real Telegram token/domain/TLS must happen on the target Linux server.
