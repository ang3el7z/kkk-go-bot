# Service Availability Matrix

Source: `docker-compose.yml`, `makefile`, `app/bot.php::menu()`, restart helpers.

| Service | Container name pattern | Mounted config/state | Probe method in PHP | Menu item | Reload/start command |
|---|---|---|---|---|---|
| `php` | `php-${VER}` | `./app:/app`, `./config:/config`, `./logs:/logs`, `/var/run/docker.sock:ro` | compose healthcheck `/check_file.sh` | bot runtime | `/start_php.sh`; webhook/runtime only |
| `service` | `service-${VER}` | `./app`, `./config`, `./logs`, `./update`, docker socket | `pgrep -f cron.php` via SSH | cron status | `/start_service.sh`; cron/check loops |
| `wg` | `wireguard-${VER}` | `config/wg0.conf`, `config/pac.json`, WG scripts | `wg` or `awg` via SSH based on PAC `amnezia` | WireGuard 0 | `wg syncconf wg0 <(wg-quick strip wg0)` or `wg-quick up/down wg0` |
| `wg1` | `wireguard1-${VER}` | `config/wg1.conf`, `config/pac.json`, WG scripts | `wg` or `awg` via SSH based on PAC `wg1_amnezia` | WireGuard 1 | same as `wg` |
| `xr` | `xray-${VER}` | `config/xray.json:/xray.json`, `logs:/logs` | `pgrep xray`; stats via `xray api stats` | Xray | `pkill xray`; `xray run -config /xray.json` |
| `np` | `naive-${VER}` | `config:/config`, `certs:/certs`, `start_np.sh` | `pgrep caddy` | NaiveProxy | `pkill caddy`; `caddy run -c /config/Caddyfile` |
| `oc` | `openconnect-${VER}` | `config:/etc/ocserv`, `certs:/certs`, `start_oc.sh` | `pgrep ocserv` | OpenConnect | `pkill ocserv`; `ocserv -c /etc/ocserv/ocserv.conf` |
| `hy` | `hysteria-${VER}` | `config:/config`, `certs:/certs`, `logs:/logs` | `pgrep hysteria` | Hysteria | `pkill hysteria`; `hysteria server -c /config/hysteria.yaml` |
| `tg` | `mtproto-${VER}` | `config/mtprotosecret`, `logs:/logs` | `pgrep mtproto-proxy` | MTProto | `pkill mtproto-proxy`; `mtproto-proxy ...` |
| `ad` | `adguard-${VER}` | `config:/config`, `certs:/certs`, `logs:/logs`, volume `adguard` | `JSON=1 timeout 2 dnslookup google.com ad` | AdGuard | `kill -15 $(cat /opt/adguardhome/pid)`; `AdGuardHome ... -c /config/AdGuardHome.yaml` |
| `ss` | `shadowsocks-${VER}` | `config/ssserver.json:/config.json` | `pgrep ssserver` | Shadowsocks | `pkill ssserver`; `ssserver -v -d -c /config.json` |
| `proxy` | `proxy-${VER}` | `config/sslocal.json:/config.json` | implicit `sslocal`/proxy route checks | Shadowsocks local proxy | `pkill sslocal`; `sslocal -v -d -c /config.json` |
| `dnstt` | `dnstt-${VER}` | `config/dnstt:/dnstt`, `logs:/logs` | `pgrep dnstt` | DNSTT | `pkill dnstt`; `dnstt-server ...` |
| `wp` | `warp-${VER}` | `config:/config`, volume `warp:/var/lib/cloudflare-warp` | `pgrep warp-svc`; `curl -x socks5://127.0.0.1:40000 ...` | Warp | `warp-svc`; `warp-cli --accept-tos connect/delete` |
| `ng` | `nginx-${VER}` | `nginx.conf`, `override.conf`, `location.conf`, certs, webapp | compose healthcheck `nginx -t`; reload via SSH | web/nginx/certs | `nginx -s reload`; `/start_ng.sh` |
| `up` | `upstream-${VER}` | `upstream.conf`, `deny`, logs | no direct dashboard probe; used for deny/upstream reload | upstream ports/domain | `nginx -s reload`; `/start_upstream.sh` |

## Availability Notes

- Main menu currently displays both process liveness and compose service presence. Go availability should persist both `enabled` from compose/config and `available` from probes.
- Task 05 seed list should include primary user-facing services `wg`, `wg1`, `xr`, `oc`, `np`, `ss`, `proxy`, `ad`, `tg`, `wp`, `dnstt`, `hy`; integration also needs support rows for `php`, `service`, `ng`, `up` because route handlers depend on them.
- Direct callbacks must check availability before mutation. Legacy PHP does not consistently guard direct callbacks; it assumes compose services are reachable by DNS service name.
