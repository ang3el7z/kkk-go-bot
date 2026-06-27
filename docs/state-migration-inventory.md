# State Migration Inventory

Source classes: DB source-of-truth, rendered file, secret, transient runtime file.

## Source Of Truth Candidates

| Path | Class | Current writers/readers | Migration note |
|---|---|---|---|
| `app/config.php` | DB source-of-truth + secret | admin list, bot token, debug, backup schedules | split token to env/secret, admins/settings to DB |
| `config/pac.json` | DB source-of-truth | central settings, domain, language, feature toggles, route lists, templates | primary settings/routes import target |
| `config/clients.json` | DB source-of-truth | WireGuard instance 0 clients/peers | import to clients + wg peer tables |
| `config/clients1.json` | DB source-of-truth | WireGuard instance 1 clients/peers | import to clients + wg peer tables |
| `config/hwid.json` | DB source-of-truth | Xray HWID device records | import to HWID/device table |
| `config/xray.stats` | DB source-of-truth | accumulated Xray traffic stats | import to stats/audit table if stats retained |
| `config/ocserv.passwd` | DB source-of-truth + secret hashes | OpenConnect users | import usernames/hashes carefully; do not log hashes |
| `config/mtprotosecret` | secret | MTProto secret | keep as secret setting, redact in reports |
| `config/mtprotodomain` | DB source-of-truth | MTProto fake domain | import as setting |
| `config/dnstt/server.key` | secret | DNSTT private key | secret material, redact |
| `config/dnstt/server.pub` | rendered/public state | DNSTT public key | can derive/export as public material |
| `certs/cert_private` | secret | TLS private key | secret material, never log |
| `certs/cert_public` | rendered/public state | TLS cert chain | import metadata/ref only unless explicit secret import |

## Rendered Config Files

| Path | Class | Owner | Migration note |
|---|---|---|---|
| `config/wg0.conf` | rendered file + runtime secret | WireGuard wg | render from DB; existing private keys must be imported as secrets |
| `config/wg1.conf` | rendered file + runtime secret | WireGuard wg1 | same as wg0 |
| `config/xray.json` | rendered file today, partial source today | Xray | clients/settings currently live here; import, then render from DB |
| `config/AdGuardHome.yaml` | rendered file today, partial source today | AdGuard | import user-managed DNS settings; render or preserve as external config |
| `config/hysteria.yaml` | rendered file + secret | Hysteria | password/port/domain settings to DB/secret, render file |
| `config/ocserv.conf` | rendered file | OpenConnect | import tun/dns/routes/expose settings; render |
| `config/Caddyfile` | rendered file | NaiveProxy | render from domain/user/pass settings |
| `config/ssserver.json` | rendered file + secret | Shadowsocks server | password/port/method settings to DB/secret |
| `config/sslocal.json` | rendered file + secret | Shadowsocks local proxy | render from SS server settings |
| `config/nginx.conf` | rendered file | nginx | render from template/domain/static paths |
| `config/upstream.conf` | rendered file | upstream nginx | render from service/domain state |
| `config/location.conf` | rendered file | nginx override | ignore if empty; preserve custom override if present |
| `config/override.conf` | rendered/custom override | nginx override | preserve as operator override, not DB-owned by default |
| `config/unit.json` | rendered file | PHP/unit web config | render from bot web routes/domain |
| `config/clash.json` | rendered template | subscriptions | keep template/import if user modified |
| `config/sing.json` | rendered template | subscriptions | keep template/import if user modified |
| `config/v2ray.json` | rendered template | subscriptions | keep template/import if user modified |
| `config/include.conf` | rendered/custom include | nginx/upstream | preserve custom include |
| `config/deny` | DB source-of-truth or rendered file | IP moderation | import deny/allow entries; render nginx deny file |
| `config/nginx_default.conf`, `config/php.ini`, `config/sshd_config`, `config/Makefile` | static config | repo | no DB import |

## Transient Runtime Files

| Path | Class | Note |
|---|---|---|
| `logs/*` | transient runtime file | searchable/clearable logs; do not import to DB except optional audit metadata |
| `update/json`, `update/branch`, `update/key`, `update/curl`, `update/pipe`, `update/message`, `update/reload_message`, `update/update_pid` | transient runtime file | updater IPC; do not import |
| PHP sessions | transient runtime file | `session_id(from)` stores reply prompts; do not import |
| `backup.json` | transient/operator export | ignored; treat as import input only |
| `override.env` | secret/operator env | do not import automatically; env source |
| `.env` | secret/operator env | required compose env; do not commit or log |
| `docker-compose.override.yml` | rendered/operator override | import only service enable/port overrides if explicitly requested |
| `ssh/key`, `ssh/key.pub` | secret/public auth material | secret; do not log |
| `singbox_windows/singbox.zip` | binary asset | no import |
| `app/webapp/override.html` | operator-uploaded web asset | preserve as file/blob, not core DB |
| `app/zapretlists/*` | downloaded/imported list cache | transient or route-list import input |
| `mirror/start_socat.sh` | static helper | no import |

## Migration Boundaries

Importer must be read-only by default and redact: Telegram token, WireGuard private keys, preshared keys, Shadowsocks/Naive/OpenConnect/Hysteria passwords, MTProto secret, DNSTT private key, TLS private key, SSH keys, password hashes unless explicitly requested.

The DB source of truth should replace JSON/YAML as write targets before Go service mutation slices. Until then, Go slice 1 should only read availability and first-slice admin/settings state.
