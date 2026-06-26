# Agent instructions

This repo is imported from `mercurykd/vpnbot` and will be maintained as `ang3el7z/kkk-go-bot`.

## Scope

- Main runtime target: Ubuntu 22.04/24.04 or Debian 11/12 server with Docker.
- Do not run destructive make targets without explicit user approval: `make delete`, `make reset`, `make dv`.
- Do not commit runtime secrets or generated state.

## Secrets and runtime files

- `app/config.php` contains the Telegram bot token and is ignored.
- `config/hwid.json` is tracked with an empty `{}` default so HWID-limit code has a safe baseline.
- `override.env`, `docker-compose.override.yml`, `backup.json`, certs, ssh keys, logs, and generated WireGuard files are ignored.
- `config/wg0.conf` and `config/wg1.conf` are touched before startup but must not be committed after runtime writes WireGuard private keys.
- Empty placeholders such as `.gitkeep` are safe.

## Development

- Prefer targeted reads/searches; `app/bot.php` is large.
- For local inspection, read `PROJECT_MAP.md` first.
- Narrow validation first:
  - `docker compose config`
  - `php -l app/init.php`
  - `php -l app/bot.php`
- Full runtime validation needs a Linux host with Docker and a real Telegram bot token.
