# Lead Task Plan: Go Rewrite

Local note: this branch did not contain `docs/lead-task-plan.md`; Task 01 source plan was found in sibling worktree `kkk-go-bot-2`. This file records validated Task 01 results and boundary corrections for Lead.

## Task 01

Title: Analyst Legacy Inventory Gate

Status: Proceed

Artifacts:

- [Legacy route inventory](legacy-route-inventory.md)
- [State migration inventory](state-migration-inventory.md)
- [Service availability matrix](service-availability-matrix.md)

Acceptance check:

- Route inventory covers `Bot::action()` message/callback patterns, dynamic reply dispatcher, handler area, mutation flag, and required service at grouped route level.
- State inventory classifies legacy JSON/YAML/conf/PHP/session/update/log files as DB source-of-truth, rendered file, secret, or transient runtime file.
- Service matrix covers compose service name, container name pattern, mounted config, probe method, menu item, and reload/start command.

## Boundary Corrections

- Task 05 service registry should seed user-facing services `wg`, `wg1`, `xr`, `oc`, `np`, `ss`, `proxy`, `ad`, `tg`, `wp`, `dnstt`, `hy`; it also needs support-service awareness for `php`, `service`, `ng`, `up` because many callbacks mutate via those services.
- Task 08 importer must treat `app/config.php` as both config source and secret file. Telegram token must remain env/secret-only in Go; admin/settings can migrate to DB.
- WireGuard and Xray later slices depend on importer redaction rules because legacy backup/export includes private keys, preshared keys, cert private material, and password hashes.
- Go runtime implementation should not start from this analyst task.

Final Analyst note: `Proceed`.
