# Go Rewrite Status

## Integrated

- Go module and Docker build target.
- SQLite migrations.
- Legacy read-only importer with secret redaction.
- Telegram webhook runtime.
- Admin bootstrap and `/id`.
- Main menu generated from service availability.
- Compose profile `go-bot`.
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
- GO-002/003/004/006/008 are partially implemented by this integration.
- GO-005/007 importer exists but needs full legacy coverage.
- GO-010 WireGuard parity implemented in Go. Runtime validation on Linux host still required.
- GO-011 Xray parity implemented in Go. Runtime validation on Linux host still required.
- GO-012 PAC/subscription parity implemented in Go. Runtime validation on Linux host still required.
- GO-009..GO-018 remain implementation work for parity/cutover.
