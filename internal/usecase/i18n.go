package usecase

var translations = map[string]map[string]string{
	"ad_title":             {"en": "AdGuard", "ru": "AdGuard"},
	"add":                  {"en": "add", "ru": "добавить"},
	"admin":                {"en": "admin", "ru": "админа"},
	"amnezia":              {"en": "Amnezia", "ru": "Amnezia"},
	"back":                 {"en": "back", "ru": "назад"},
	"backup":               {"en": "auto backup", "ru": "автобэкап"},
	"chat":                 {"en": "chat", "ru": "чат поддержки"},
	"config":               {"en": "Settings", "ru": "настройки"},
	"delete":               {"en": "delete", "ru": "удалить"},
	"donate":               {"en": "donate", "ru": "донат"},
	"domain explain":       {"en": "Some clients require a valid certificate when connecting, such as windows 11 DoH or ShadowSocks Android (PAC url), this requires a domain", "ru": "Некоторым клиентам требуется валидный сертификат при подключении, например Windows 11 DoH или ShadowSocks Android (URL-адрес PAC), для этого требуется домен"},
	"export":               {"en": "backup", "ru": "сохранить"},
	"import":               {"en": "restore", "ru": "восстановить"},
	"install domain":       {"en": "install domain", "ru": "установить домен"},
	"lang":                 {"en": "language", "ru": "язык интерфейса"},
	"Letsencrypt SSL":      {"en": "Letsencrypt SSL", "ru": "Letsencrypt SSL"},
	"logs":                 {"en": "logs", "ru": "логи"},
	"mtproto":              {"en": "MTProto", "ru": "MTProto"},
	"naive":                {"en": "NaiveProxy", "ru": "NaiveProxy"},
	"ocserv":               {"en": "OpenConnect", "ru": "OpenConnect"},
	"off":                  {"en": statusOff, "ru": statusOff},
	"on":                   {"en": statusOn, "ru": statusOn},
	"pac":                  {"en": "PAC", "ru": "PAC"},
	"page":                 {"en": "pagination", "ru": "пагинация"},
	"restart":              {"en": "restart", "ru": "перезагрузка"},
	"Self SSL":             {"en": "Self SSL", "ru": "Собственный SSL"},
	"sh_title":             {"en": "Shadowsocks", "ru": "Shadowsocks"},
	"warp":                 {"en": "Warp", "ru": "Warp"},
	"wg_title":             {"en": "Wireguard", "ru": "Wireguard"},
	"xray":                 {"en": "Vless", "ru": "Vless"},
	"container management": {"en": "container management", "ru": "управление контейнерами"},
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
