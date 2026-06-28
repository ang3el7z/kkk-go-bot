# Menu Parity Audit

Source: `app/bot.php` from `mercurykd/vpnbot` legacy import.
Target: Go usecase menus in `internal/usecase`.

Goal: keep user-visible texts, entry points, menu order, and behavior close to upstream, while using Go-native callbacks and services.

## Top-Level Menu

| Entry | Upstream function | Go status | Remaining gap |
|---|---:|---|---|
| WireGuard / WireGuard 1 | `statusWg` | Layout mostly aligned; Go-native actions for add/toggle/delete/QR/download/DNS/MTU/timer/AllowedIPs, default DNS/MTU, amnezia, torrent, exchange, subnets | Exact pagination and status formatting still differs from PHP |
| Vless | `xray` | Upstream-like top rows and Go-native client actions | Main outbound, reset monthly, IP limit, fake domain, template chooser/download, route submenus, detailed user page |
| NaiveProxy | `naiveMenu` | Upstream-like shell | Change subdomain/login/password and download logic not migrated |
| OpenConnect | `ocMenu` | Upstream-like shell | User CRUD, secret/password/DNS/subnet/expose logic not migrated |
| MTProto | `mtproto` | Upstream-like shell | Generate/set secret, fake domain, QR logic not migrated |
| AdGuard | `adguardMenu` | Basic Go menu only | Web panel/browser toggle/password/ClientID/allowed clients/check DNS/reset/upstream layout |
| Warp | `warp` | Upstream-like shell | Toggle Warp+, key, per-client subscriptions/templates/QR/HWID/user actions |
| Shadowsocks | `menuSS` | Upstream-like shell | Password, v2ray toggle, QR logic not migrated |
| PAC | `pacMenu` | Basic subscription list | Upstream PAC/reverse PAC web-app links and antifilter lists |
| Hysteria | `hysteriaMenu` | Upstream-like shell | Password/download logic not migrated |
| DNSTT | `dnstt` | Upstream-like shell | Show-in-menu toggle, pubkey download, domain/password logic |
| Settings | `configMenu` | Layout mostly aligned plus container management | Domain/SSL/lang/page/autoupdate/update/restart/admin actions partial |
| Ports | `ports` | Go container-management toggles | Original hide/change port semantics only partly mapped |
| Logs / IP ban | `logs`, `ipMenu` | Basic moderation menu | Original log browser, suspicious IP analysis, deny/white lists, auto-clean |

## Callback Policy

Do not port PHP callback strings directly. Use Go namespaces:

- `service:*` for top-level routing.
- `wg:*`, `xray:*`, `ad:*`, `mod:*`, `backup:*` for migrated modules.
- `settings:*` for visible-but-not-yet-migrated settings routes.
- `svc:*` for container management.

Visible buttons may exist before full backend migration, but they must route to a controlled Go response, not an unknown/broken callback.

## Next Migration Batches

1. AdGuard parity: web panel button, browser toggle, password/key prompts, allowed clients, check DNS, reset, upstream list.
2. PAC parity: exact PAC/reverse PAC/web-app links and antifilter list management.
3. OpenConnect/Naive/Hysteria config actions: prompts, config writes, container restart.
4. Settings actions: domain/SSL/lang/pagination/autoupdate/branches/restart/admin.
5. Logs/IP ban parity: log list, search, clean, deny/white list, suspicious IP analysis.
