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

## Autostart

```sh
crontab -e
```

Add:

```sh
@reboot cd /root/kkk-go-bot && make r
```

