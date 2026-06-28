# kkk-go-bot

Fork/import of `mercurykd/vpnbot` prepared for local agent-based maintenance.

## First server install

On Ubuntu/Debian server:

```sh
git clone https://github.com/ang3el7z/kkk-go-bot.git /root/kkk-go-bot
cd /root/kkk-go-bot
echo "<?php

\$c = ['key' => 'YOUR_TELEGRAM_BOT_KEY'];" > ./app/config.php
make u
```

## Restart

```sh
make r
```

## Optional Go runtime

PHP remains the default runtime. To run the Go bot side-by-side, put the token in `.env`:

```sh
TELEGRAM_BOT_TOKEN=YOUR_TELEGRAM_BOT_KEY
```

Start/stop only the Go container:

```sh
make go
make go-down
```

Go listens on `127.0.0.1:8082` and uses compose profile `go-bot`. Rollback is `make go-down`; PHP containers stay unchanged.
Telegram long polling is enabled by default, so a test server does not need a public HTTPS webhook. Set `TELEGRAM_POLLING=0` when running behind an HTTPS webhook.

Final cutover/rollback runbook:

```sh
make go-cutover
make go-rollback
```

See `docs/go-cutover-runbook.md`.

## Autostart

```sh
crontab -e
```

Add:

```sh
@reboot cd /root/kkk-go-bot && make r
```
