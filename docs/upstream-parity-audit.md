# Upstream Parity Audit

Source: `mercurykd/vpnbot` `app/bot.php`.

## Product Rule

The Go rewrite must preserve the upstream Telegram surface unless a change is explicitly intentional:

- same primary menu layout;
- same visible service names;
- same legacy `callback_data` routes where practical;
- same feature behavior backed by DB instead of JSON/runtime files;
- hidden buttons only when the related service/container is disabled or unavailable.

## Current Gap Summary

- Upstream exposes about 214 callback route variants in `app/bot.php`.
- Current Go implementation covers the core service paths, but not full Telegram parity.
- The previous Go menu used new callbacks such as `wg:*` and `xray:*`; upstream uses callbacks such as `/changeWG 0`, `/xray`, `/menu naive`, `/menu config`.

## Main Menu Parity

Upstream main menu order:

1. `Wireguard` / `Wireguard`
2. `Vless` / `NaiveProxy`
3. `OpenConnect` / `MTProto`
4. `AdGuard` / `Warp`
5. `Shadowsocks` / `PAC`
6. `Hysteria` / `DNSTT`
7. `Settings`
8. `chat` / `donate`

Go keeps this order and hides service buttons when the service is not enabled+available.

## Remaining Parity Work

- WireGuard submenu: align visible rows/text/callbacks with `/changeWG`, `/menu wg`, `/add`, `/switchClient`, `/qr`, `/download`, `/delete`, `/rename`, `/timer`, `/subnet`.
- Xray submenu: align `/xray`, `/userXr`, `/addXrUser`, `/listXr`, `/switchXr`, `/timerXr`, `/templates`, `/routes`, HWID, transport, stats.
- Settings submenu: restore upstream settings surface: ports, logs, IP deny/allow, language, export/import/backup, update, restart, admins, SSL/domain.
- PAC/routing menus: restore `/pacMenu`, `/routes`, route list editors, template download/default/copy/add/delete.
- Smaller service menus: replace summary-only screens with upstream mutation flows for MTProto, NaiveProxy, OpenConnect, Shadowsocks, Hysteria, DNSTT, Warp.
- Message text/i18n: restore upstream Russian/English labels instead of new English-only text.
