# Go Rewrite Status

## Integrated

- Go module and Docker build target.
- SQLite migrations.
- Legacy read-only importer with secret redaction.
- Telegram webhook runtime.
- Admin bootstrap and `/id`.
- Main menu generated from service availability.
- Compose profile `go-bot`.
- WireGuard first parity slice: DB-backed clients/server config, key generation, add/delete/toggle, reply flows for rename/timer/DNS/MTU/AllowedIPs, Telegram config document upload, `wg0.conf`/`wg1.conf` rendering, optional reload via `WG_RELOAD=1`.

## Remaining For Full Parity

- WireGuard remaining: subnet list editor, QR image upload, Amnezia toggles, richer status/traffic display.
- Xray use cases: users, stats, routes, templates, HWID.
- PAC/routing list editors and remote list updates.
- Service-specific handlers: AdGuard, MTProto, SS, OC, Naive, Hysteria, DNSTT, Warp.
- Logs/IP moderation/updater/backup/import-export.
- DB-owned renderers for config files and reload commands.

## GitHub Issue Mapping

- GO-001 can close after inventory docs are merged.
- GO-002/003/004/006/008 are partially implemented by this integration.
- GO-005/007 importer exists but needs full legacy coverage.
- GO-010 has first working slice but is not full legacy parity yet.
- GO-009..GO-018 remain implementation work for parity/cutover.
