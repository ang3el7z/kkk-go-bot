# Go Rewrite Status

## Integrated

- Go module and Docker build target.
- SQLite migrations.
- Legacy read-only importer with secret redaction.
- Telegram webhook runtime.
- Admin bootstrap and `/id`.
- Main menu generated from service availability.
- Compose profile `go-bot`.

## Remaining For Full Parity

- WireGuard use cases: create/delete/rename/timer/DNS/MTU/subnets/QR/download/render/reload.
- Xray use cases: users, stats, routes, templates, HWID.
- PAC/routing list editors and remote list updates.
- Service-specific handlers: AdGuard, MTProto, SS, OC, Naive, Hysteria, DNSTT, Warp.
- Logs/IP moderation/updater/backup/import-export.
- DB-owned renderers for config files and reload commands.

## GitHub Issue Mapping

- GO-001 can close after inventory docs are merged.
- GO-002/003/004/006/008 are partially implemented by this integration.
- GO-005/007 importer exists but needs full legacy coverage.
- GO-009..GO-018 remain implementation work for parity/cutover.
