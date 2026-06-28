# Go Rewrite Status

## Integrated

- Go module and Docker build target.
- Go skeleton: `cmd/kkk-go-bot` entrypoint plus `internal/app`, `config`, `storage`, `legacy`, `services`, `telegram`, `usecase`, `wireguard`, and `xray` packages.
- SQLite migrations.
- SQLite migration runner: versioned `schema_migrations`, idempotent table creation, and indexes for menu services, clients, pending operations, and secret settings.
- Legacy read-only importer with secret redaction.
- Legacy state importer: PAC, WG clients/server configs, HWID, Xray users/stats, subscription templates, service configs, deny/include files, MTProto/DNSTT/cert/OC/SS/Naive/Hysteria secret settings.
- Legacy config bridge: optional `app/config.php` import for admins and non-secret scalar settings, with Telegram token/password/secret values redacted and kept out of Go runtime config.
- Telegram webhook runtime.
- Telegram adapter: update models, message/callback dispatch, HTML messages, inline keyboards, document/photo uploads, callback answers, and API error validation.
- Admin bootstrap and `/id`.
- Main menu generated from service availability.
- Telegram menu builder: reusable inline keyboard row/column builder used by dashboard menus, with callback and URL button support.
- Feature flags/service availability: compose + Docker service registry controls visible menu entries and blocks direct callbacks/messages for unavailable WG/Xray services.
- Compose profile `go-bot`.
- Go runtime contract: `bot` compose profile with SQLite data volume, writable config volume for renderers, read-only Docker socket for probes/exec, legacy PHP config import mount, sing-box Windows assets mount, and HTTP healthcheck.
- WireGuard parity slice: DB-backed clients/server config, key generation, add/delete/toggle, reply flows for rename/timer/DNS/MTU/AllowedIPs/default AllowedIPs, Telegram config document upload, QR image upload, Amnezia toggle/key material, endpoint switch, torrent/exchange flags synced to `pac.json`, subnet list UI, live traffic/handshake display via Docker exec, richer menu status, `wg0.conf`/`wg1.conf` rendering, optional reload via `WG_RELOAD=1`.
- Xray parity slice: import users from `xray.json`, DB-backed add/delete/toggle/rename/timer/reset UUID, stats display from `xray.stats`, full stats ingestion loop through Docker exec, VLESS link + QR, reset-stats marker, transport switching, global/per-user HWID controls, route lists, subscription templates, render active users back to `config/xray.json`, Telegram menu callbacks.
- PAC/subscription parity slice: `/pac?s=<uuid>&t=s|si|cl`, `/pac/sub?id=<uuid>`, and legacy `/pacHASH/<base64-php-params>` URLs return DB-backed Xray subscriptions; origin/DB templates, route placeholders, app import redirects, and Windows sing-box ZIP are implemented.

## Remaining For Full Parity

- PAC/routing list editors and remote list updates.
- Service-specific handlers: AdGuard, MTProto, SS, OC, Naive, Hysteria, DNSTT, Warp.
- Logs/IP moderation/updater/backup/import-export.
- DB-owned renderers for config files and reload commands.

## GitHub Issue Mapping

- GO-001 can close after inventory docs are merged.
- GO-002 Compose/runtime contract implemented.
- GO-003 Go skeleton implemented.
- GO-004 Telegram adapter implemented.
- GO-006 SQLite migrations implemented.
- GO-008 Service availability and feature flags implemented.
- GO-009 Telegram menu builder implemented.
- GO-005 Legacy config bridge implemented.
- GO-007 legacy state import implemented.
- GO-010 WireGuard parity implemented in Go. Runtime validation on Linux host still required.
- GO-011 Xray parity implemented in Go. Runtime validation on Linux host still required.
- GO-012 PAC/subscription parity implemented in Go. Runtime validation on Linux host still required.
- GO-009..GO-018 remain implementation work for parity/cutover.
