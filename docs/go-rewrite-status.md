# Go Rewrite Status

## Integrated

- Go module and Docker build target.
- SQLite migrations.
- Legacy read-only importer with secret redaction.
- Telegram webhook runtime.
- Admin bootstrap and `/id`.
- Main menu generated from service availability.
- Compose profile `go-bot`.
- WireGuard first parity slice: DB-backed clients/server config, key generation, add/delete/toggle, reply flows for rename/timer/DNS/MTU/AllowedIPs/default AllowedIPs, Telegram config document upload, QR image upload, Amnezia toggle/key material, richer menu status, `wg0.conf`/`wg1.conf` rendering, optional reload via `WG_RELOAD=1`.
- Xray first parity slice: import users from `xray.json`, DB-backed add/delete/toggle/rename/timer, VLESS link + QR, reset-stats marker, render active users back to `config/xray.json`, Telegram menu callbacks.
- PAC/subscription first slice: `/pac?s=<uuid>&t=s|si|cl` returns VLESS, sing-box JSON, or Clash YAML from DB-backed Xray users.

## Remaining For Full Parity

- WireGuard remaining: live traffic/handshake display from running WG containers.
- Xray remaining: full stats ingestion/display, routes, templates, HWID, transport switching, reset-user UUID flow.
- PAC/subscription remaining: legacy URL compatibility, templates, app redirect imports, Windows sing-box ZIP.
- PAC/routing list editors and remote list updates.
- Service-specific handlers: AdGuard, MTProto, SS, OC, Naive, Hysteria, DNSTT, Warp.
- Logs/IP moderation/updater/backup/import-export.
- DB-owned renderers for config files and reload commands.

## GitHub Issue Mapping

- GO-001 can close after inventory docs are merged.
- GO-002/003/004/006/008 are partially implemented by this integration.
- GO-005/007 importer exists but needs full legacy coverage.
- GO-010 has first working slice but is not full legacy parity yet.
- GO-011 has first working slice but is not full legacy parity yet.
- GO-012 has first working slice but is not full legacy parity yet.
- GO-009..GO-018 remain implementation work for parity/cutover.
