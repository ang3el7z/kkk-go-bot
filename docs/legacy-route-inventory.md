# Legacy Route Inventory

Source: `app/bot.php`, `Bot::input()`, `Bot::auth()`, `Bot::session()`, `Bot::action()`.

## Global Gates

| Flow | Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|---|
| `/id` message | `^/id$` before auth and in `action()` | `send()` | identity | no | php |
| first non-`/id` user | any message/callback when `app/config.php` has no `admin` | `auth()` | admin bootstrap | yes, `app/config.php` | php |
| unauthorized user | any non-admin message/callback | `auth()` exits | auth | no | php |
| force reply | any reply to bot prompt | `reply()` dynamic callback from `$_SESSION['reply']` | multi-step input | usually yes | php plus handler service |

`Bot::action()` is a single `switch (true)` over message text, callback data, and reply state. Runtime Go must preserve callback strings for Telegram inline buttons or provide a compatibility adapter.

## Main And Admin Routes

| Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|
| message `/start`, `/menu`; callback `/menu` | `menu()` | dashboard | no | php, service, compose targets for probes |
| callback `/menu addpeer <n>`, `/menu wg <n>`, `/menu client <id_page>` | `menu()` | WireGuard menus | no | wg or wg1 |
| callback `/menu pac|adguard|config|ss|lang|oc|naive|mirror|update|hy` | `menu()` | section menus | no | section service |
| message `/mirror` | `menu('mirror')` | mirror | no | php |
| callback `/debug` | `debug()` | diagnostics | yes, `app/config.php` debug flag | php |
| callback `/backup` | `backup()` -> reply `setBackup()` | auto backup schedule | yes, `app/config.php` | service |
| callback `/autoCleanLogs` | `autoCleanLogs()` -> reply `setAutoCleanLogs()` | logs | yes, `app/config.php` | service |
| callback `/logs`, `/getLog`, `/clearLog`, `/cleanLog`, `/delLog` | log handlers | logs | clear/delete routes mutate `/logs/*` | service/log volume |
| callback `/restart`, `/applyupdatebot`, `/branches`, `/changeBranch <n>` | update/restart handlers | updater | yes, `/update/*`, git checkout/pull | service |
| callback `/ports`, `/hidePort <svc>`, reply `setPort()` | port config | compose override | yes, `docker-compose.override.yml` | service/docker socket |
| callback `/export`, `/import`, `/importList <type>` | backup/import handlers | migration/import | yes on import | php plus all configured services |

## WireGuard Routes

| Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|
| callback `/changeWG <0|1>` | `changeWG()` | instance selector | session only | php |
| callback `/add`, `/add_ips` | `addPeer()`, `addips()` -> `createPeer()` | peer create | yes, clients JSON + WG conf | wg or wg1 |
| callback `/delete <client_page>`, `/switchClient <client_page>` | `deletePeer()`, `switchClient()` | peer lifecycle | yes | wg or wg1 |
| callback `/rename <client_page>`, `/timer <client_page>` | reply `renameClient()`, `timerClient()` | peer metadata | yes | wg or wg1 |
| callback `/dns`, `/deletedns`, `/defaultDNS`, `/changeMTU`, `/defaultMTU` | DNS/MTU handlers | peer config | yes | wg or wg1 |
| callback `/subnet`, `/subnetAdd`, `/subnetDelete`, `/addSubnets`, `/changeAllowedIps`, `/changeIps` | subnet/IP handlers | routing | yes | wg or wg1 |
| callback `/switchTorrent`, `/switchExchange`, `/switchEndpoint`, `/switchAmnezia`, `/resetAmnezia`, `/blinkmenuswitch` | WG mode toggles | WG feature flags | yes | wg or wg1 |
| callback `/qr <id>`, `/download <id>`, `/dw wg <id>` | QR/download | export | temp QR/write only | php, wg/wg1 for config read |

State touched: `/config/clients.json`, `/config/clients1.json`, `/config/wg0.conf`, `/config/wg1.conf`, `/config/pac.json`; reload via SSH to `wg`/`wg1`.

## Xray, PAC, Routing, HWID

| Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|
| callback `/xray`, `/userXr`, `/addXrUser`, `/renameXrUser`, `/resetXrUser`, `/switchXr`, `/delxr`, `/listXr`, `/timerXr` | Xray user handlers | Xray users | yes | xr |
| callback `/qrXray`, `/v2ray` | link/export | Xray subscriptions | no | xr/php |
| callback `/resetXrStats` | `resetXrStats()` | stats | yes, `/config/xray.stats` | xr |
| callback `/changeTransport`, `/changeFakeDomain`, `/selfFakeDomain`, `/mainOutbound` | Xray/PAC transport | routing config | yes | xr, up, ng |
| callback `/xtlsblock`, `/routes`, `/xtlswarp`, `/xtlsproxy`, `/xtlsapp`, `/xtlsprocess`, `/xtlssubnet`, `/xtlsrulesset` | route-list menus | split routing | no | php |
| callback `/include`, `/exclude`, `/reverse`, `/subzones`, `/delete|change<typelist>` | PAC/rule list editors | split routing | yes | php/xr |
| callback `/paczapret`, `/pacupdate`, `/addCommunityFilter`, `/addLegizFilter` | PAC updater | split routing | yes, may fetch remote lists | php/xr |
| callback `/hwidLimit`, `/toggleHwidLimit`, `/setHwidDevices`, `/hwidUser*`, `/setHwidUserLimit` | HWID handlers | Xray HWID limits | yes | xr/php |
| callback `/templates`, `/templateAdd`, `/templateCopy`, `/delTemplate`, `/downloadOrigin`, `/downloadTemplate`, `/defaultTemplate`, `/choiceTemplate`, `/templateUser` | config templates | Xray/PAC templates | yes for add/copy/delete/default | php |

State touched: `/config/xray.json`, `/config/xray.stats`, `/config/hwid.json`, `/config/pac.json`, `/config/clash.json`, `/config/sing.json`, route-set temp files; reload via SSH to `xr` and upstream/nginx helpers.

## Service-Specific Routes

| Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|
| callback `/adguardChBr`, `/adguardpsswd`, `/setAdguardKey`, `/adguardreset`, `/addupstream`, `/adgFillAllowedClients`, `/checkdns` | AdGuard handlers | DNS/filtering | yes except probes | ad |
| callback `/mtproto`, `/generateSecret`, `/setSecret`, `/changeTGDomain`, `/qrMtproto` | MTProto handlers | Telegram proxy | yes except display/QR | tg |
| callback `/sspswd`, `/qrSS` | Shadowsocks handlers | SS server/client | yes for password | ss, proxy |
| callback `/changeOcDomain`, `/changeOcPass`, `/changeOcDns`, `/changeOcExpose`, `/addOcUser`, `/deloc` | OpenConnect handlers | OC users/config | yes | oc, up |
| callback `/changeNaiveUser`, `/changeNaiveSubdomain`, `/changeNaivePass` | NaiveProxy handlers | Naive config | yes | np, up |
| callback `/changeHysteriaPass`, `/hy menu` | Hysteria handlers | Hysteria config | yes for password | hy |
| callback/message `/dnstt`, `/showdnstt`, `/dnsttDownload`, `/dnsttDomain`, `/dnsttPassword`, `/setdnsttDomain`, `/setdnsttPassword` | DNSTT handlers | DNSTT SSH tunnel | yes for domain/password/key generation | dnstt |
| callback `/warp`, `/warpPlus`, `/offWarp` | Warp handlers | Cloudflare Warp | yes | wp |
| callback `/selfssl`, `/setSSL <name>`, `/deletessl`, `/domain`, `/deldomain`, `/addNipdomain`, `/addSubdomain`, `/addLinkDomain` | domain/TLS handlers | certs/nginx/unit | yes | ng, up, oc, np, ad |
| callback `/proxy`, `/appOutbound`, `/domainsOutbound`, `/finalOutbound`, `/processOutbound`, `/offWarp` | routing/outbound handlers | proxy/Warp routing | yes | proxy, wp, xr |
| callback `/mirror`, `/getMirror`, message `/mirror` | mirror handlers | local mirror | yes for generated mirror config | php/ng |

## IP And Log Moderation Routes

| Pattern | Handler | Area | Mutates | Required service |
|---|---|---|---|---|
| callback `/switchBanIp`, `/switchMonthlyStats`, `/setIpLimit`, `/switchSilence`, `/switchScanIp`, `/autoScanTimeout` | settings toggles | abuse controls | yes, `/config/pac.json` | php |
| callback `/ipMenu`, `/analysisIp`, `/searchIp`, `/searchSuspiciousIp`, message/callback `/searchLogs ...` | analysis/search | logs | no | logs volume |
| callback `/cleanDeny`, `/denyList`, `/denyIp`, `/whiteIp`, `/allowIp`, `/cleanLogs` | deny/allow/log cleanup | nginx deny list | yes, `/config/deny`, `/logs/*` | up |
| callback `/importIps <source>` | `importIps()` | provider CIDRs | yes, `/config/pac.json` lists | network/php |

## Reply Callback Boundary

Reply prompts dynamically dispatch to method names stored in `$_SESSION['reply'][message_id]['callback']`. Important callbacks include: `secretSet`, `urlcheck`, `sspwdch`, `chockey`, `chOcSubdomain`, `chocdns`, `chocpass`, `chnplogin`, `chNpSubdomain`, `chnppass`, `chhypass`, `addocus`, `addxrus`, `renXrUs`, `setOverrideHtml`, `renameClient`, `timerClient`, `importListFile`, `importFile`, `createPeer`, `addDomain`, `selfsslInstall`, `chpsswd`, `setAdKey`, `setTimerXr`, `addAdmin`, `setSubdomain`, `switchIpLimit`, `setPort`, `setLinkDomain`, `setPage`, `dnscheck`, `upstream`, `addInclude`, `addReverse`, `addSubzones`, `addExclude`, `setDNS`, `setMTU`, `changeClientMTU`, `subnetSave`, `setIps`, `setdnsttDomain`, `setdnsttPassword`, `saveHwidDevices`, `saveHwidUserLimit`, `addTemplate`, `copyTemplate`, `setAutoScanTimeout`, `setMainOutbound`, `addWarpPlus`, `setBackup`, `setAutoCleanLogs`, `setFakeDomain`, `setTelegramDomain`.

Go rewrite boundary: model reply flows as explicit pending operations, not arbitrary method dispatch.
