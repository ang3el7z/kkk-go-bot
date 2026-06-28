package usecase

var translations = map[string]map[string]string{
	"ad_title":               {"en": "AdGuard", "ru": "AdGuard"},
	"add":                    {"en": "add", "ru": "добавить"},
	"admin":                  {"en": "admin", "ru": "админа"},
	"amnezia":                {"en": "Amnezia", "ru": "Amnezia"},
	"back":                   {"en": "back", "ru": "назад"},
	"backup":                 {"en": "auto backup", "ru": "автобэкап"},
	"chat":                   {"en": "chat", "ru": "чат поддержки"},
	"cdn":                    {"en": "cdn", "ru": "cdn"},
	"changeFakeDomain":       {"en": "changeFakeDomain", "ru": "фейковый домен"},
	"change login":           {"en": "change login", "ru": "изменить логин"},
	"change password":        {"en": "change password", "ru": "изменить пароль"},
	"change secret":          {"en": "change secret", "ru": "секретное слово"},
	"change subdomain":       {"en": "change subdomain", "ru": "изменить поддомен"},
	"config":                 {"en": "Settings", "ru": "настройки"},
	"delete":                 {"en": "delete", "ru": "удалить"},
	"donate":                 {"en": "donate", "ru": "донат"},
	"domain explain":         {"en": "Some clients require a valid certificate when connecting, such as windows 11 DoH or ShadowSocks Android (PAC url), this requires a domain", "ru": "Некоторым клиентам требуется валидный сертификат при подключении, например Windows 11 DoH или ShadowSocks Android (URL-адрес PAC), для этого требуется домен"},
	"download":               {"en": "download", "ru": "скачать"},
	"download pubkey":        {"en": "download pubkey", "ru": "скачать pubkey"},
	"dns":                    {"en": "dns", "ru": "днс"},
	"exchange":               {"en": "client isolation", "ru": "изоляция клиентов"},
	"export":                 {"en": "backup", "ru": "сохранить"},
	"generateSecret":         {"en": "generateSecret", "ru": "сгенерировать ключ"},
	"add peer":               {"en": "add peer", "ru": "добавить клиента"},
	"defaultDNS":             {"en": "defaultDNS", "ru": "дефолтный днс"},
	"defaultMTU":             {"en": "defaultMTU", "ru": "дефолтный MTU"},
	"ip limit":               {"en": "ip limit", "ru": "лимит ip"},
	"import":                 {"en": "restore", "ru": "восстановить"},
	"install domain":         {"en": "install domain", "ru": "установить домен"},
	"lang":                   {"en": "language", "ru": "язык интерфейса"},
	"Letsencrypt SSL":        {"en": "Letsencrypt SSL", "ru": "Letsencrypt SSL"},
	"listSubnet":             {"en": "listSubnet", "ru": "список подсетей"},
	"logs":                   {"en": "logs", "ru": "логи"},
	"main outbound name: ":   {"en": "main outbound name: ", "ru": "имя основного outbound: "},
	"mtproto":                {"en": "MTProto", "ru": "MTProto"},
	"naive":                  {"en": "NaiveProxy", "ru": "NaiveProxy"},
	"ocserv":                 {"en": "OpenConnect", "ru": "OpenConnect"},
	"off":                    {"en": statusOff, "ru": statusOff},
	"on":                     {"en": statusOn, "ru": statusOn},
	"pac":                    {"en": "PAC", "ru": "PAC"},
	"page":                   {"en": "pagination", "ru": "пагинация"},
	"rename":                 {"en": "rename", "ru": "переименовать"},
	"reset monthly":          {"en": "reset monthly", "ru": "ежемесячный сброс"},
	"reset stats":            {"en": "reset stats", "ru": "сбросить статистику"},
	"restart":                {"en": "restart", "ru": "перезагрузка"},
	"Self SSL":               {"en": "Self SSL", "ru": "Собственный SSL"},
	"sh_title":               {"en": "Shadowsocks", "ru": "Shadowsocks"},
	"show QR":                {"en": "show QR", "ru": "показать QR"},
	"selfFakeDomain":         {"en": "steal from yourself", "ru": "steal from yourself"},
	"set key":                {"en": "set key", "ru": "установить ключ"},
	"set hwid devices count": {"en": "set HWID devices count", "ru": "установить количество HWID устройств"},
	"set password":           {"en": "set password", "ru": "установить пароль"},
	"set subdomain":          {"en": "set subdomain", "ru": "установить поддомен"},
	"setSecret":              {"en": "setSecret", "ru": "установить свой ключ"},
	"show in menu ":          {"en": "show in menu ", "ru": "показывать в меню "},
	"v2ray templates":        {"en": "v2ray templates", "ru": "шаблоны v2ray"},
	"sing-box templates":     {"en": "sing-box templates", "ru": "шаблоны sing-box"},
	"mihomo templates":       {"en": "mihomo templates", "ru": "шаблоны mihomo"},
	"routes":                 {"en": "routes", "ru": "маршруты"},
	"timer":                  {"en": "set timer", "ru": "установить время действия"},
	"torrent":                {"en": "torrents", "ru": "торренты"},
	"update status":          {"en": "update status", "ru": "обновить статус"},
	"warp":                   {"en": "Warp", "ru": "Warp"},
	"wg_title":               {"en": "Wireguard", "ru": "Wireguard"},
	"xray":                   {"en": "Vless", "ru": "Vless"},
	"container management":   {"en": "container management", "ru": "управление контейнерами"},
}

func i18n(pac map[string]any, key string) string {
	lang := stringValue(pac["language"])
	if lang == "" {
		lang = "en"
	}
	if values, ok := translations[key]; ok {
		if value := values[lang]; value != "" {
			return value
		}
		if value := values["en"]; value != "" {
			return value
		}
	}
	return key
}
