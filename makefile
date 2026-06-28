b:
	docker compose build
init:
	touch ./override.env ./docker-compose.override.yml ./config/location.conf ./config/override.conf ./config/hwid.json ./config/wg0.conf ./config/wg1.conf
u: # запуск контейнеров
	$(eval IP := $(shell hostname -I | awk '{print $$1}'))
	bash ./update/update.sh &
	make init
	IP=$(IP) VER=$(shell git describe --tags) docker compose --env-file ./.env --env-file ./override.env up -d --force-recreate
go: # запуск Go runtime рядом с PHP через compose profile go-bot
	make init
	COMPOSE_PROFILES=go-bot IP=$${IP:-$$(hostname -I | awk '{print $$1}')} VER=$${VER:-$$(git describe --tags 2>/dev/null || echo dev)} docker compose --env-file ./.env --env-file ./override.env up -d --force-recreate bot
go-down: # остановка только Go runtime
	COMPOSE_PROFILES=go-bot docker compose stop bot
go-logs: # логи Go runtime
	COMPOSE_PROFILES=go-bot docker compose logs -f bot
go-shell: # shell Go контейнера
	COMPOSE_PROFILES=go-bot docker compose exec bot /bin/sh
d: # остановка контейнеров
	-kill -9 $(shell cat ./update/update_pid) > /dev/null
	docker compose down --remove-orphans
dv: # остановка контейнеров
	docker compose down -v
r: d u
ps: # список контейнеров
	docker compose ps
l: # логи из контейнеров
	docker compose logs
php: # консоль сервиса
	docker compose exec php /bin/sh
wg: # консоль сервиса
	docker compose exec wg /bin/sh
wg1: # консоль сервиса
	docker compose exec wg1 /bin/sh
ss: # консоль сервиса
	docker compose exec ss /bin/sh
ng: # консоль сервиса
	docker compose exec ng /bin/sh
np: # консоль сервиса
	docker compose exec np /bin/sh
up: # консоль сервиса
	docker compose exec up /bin/sh
ad: # консоль сервиса
	docker compose exec ad /bin/sh
wp: # консоль сервиса
	docker compose exec wp bash
proxy: # консоль сервиса
	docker compose exec proxy /bin/sh
tg: # консоль сервиса
	docker compose exec tg /bin/sh
dnstt: # консоль сервиса
	docker compose exec dnstt /bin/sh
hy: # консоль сервиса
	docker compose exec hy /bin/sh
xr: # консоль сервиса
	docker compose exec xr /bin/sh
oc: # консоль сервиса
	docker compose exec oc /bin/sh
service: # консоль сервиса
	docker compose exec service /bin/sh
delete:
	make d
	docker system prune -f -a
	docker volume prune -f -a
	rm -rf /root/kkk-go-bot
push:
	docker compose push
s:
	git status -su
c:
	git add config/
	git checkout .
	git reset
webhook:
	docker compose exec php php checkwebhook.php
reset:
	make d
	git reset --hard
	git clean -fd
	docker volume rm kkk-go-bot_adguard kkk-go-bot_warp
	make u
backup:
	docker compose exec php php backup.php > backup.json
cron: # установка задачи в cron для автозапуска при перезагрузке
	@(crontab -l 2>/dev/null | grep -v "cd /root/kkk-go-bot && make r"; echo "@reboot cd /root/kkk-go-bot && make r") | crontab -
uncron: # удаление задачи из cron
	@crontab -l 2>/dev/null | grep -v "cd /root/kkk-go-bot && make r" | crontab -
