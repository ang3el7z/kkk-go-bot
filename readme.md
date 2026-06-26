telegram bot to manage servers (inside the bot)

- VLESS (Reality OR Websocket)
- NaiveProxy
- OpenConnect
- Wireguard
- Amnezia
- AdguardHome
- MTProto
- PAC
- automatic ssl

---
environment: ubuntu 22.04/24.04, debian 11/12

## Install:

```shell
wget -O- https://raw.githubusercontent.com/ang3el7z/kkk-go-bot/main/scripts/init.sh | sh -s YOUR_TELEGRAM_BOT_KEY main
```
#### Restart:
```shell
make r
```
#### autoload:
```shell
crontab -e
```
add `@reboot cd /root/kkk-go-bot && make r` and save
