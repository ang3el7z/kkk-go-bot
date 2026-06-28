package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
)

const (
	statusOn  = "🟢"
	statusOff = "🔴"
)

func (b *Bot) mainMenuText(ctx context.Context, services []storage.Service) string {
	pac := b.legacyPAC(ctx)
	serviceByName := servicesByName(services)
	lines := []string{"v" + envDefault("VER", "dev") + branchSuffix()}
	if domain := stringValue(pac["domain"]); domain != "" {
		lines = append(lines, "", "<blockquote>", "Domains:", html.EscapeString(domain))
		if np := stringValue(pac["np_domain"]); np != "" {
			lines = append(lines, "naive "+html.EscapeString(np+"."+domain))
		}
		if oc := stringValue(pac["oc_domain"]); oc != "" {
			lines = append(lines, "openconnect "+html.EscapeString(oc+"."+domain))
		}
		if ad := stringValue(pac["adguardkey"]); ad != "" {
			lines = append(lines, html.EscapeString(ad+"."+domain)+" adguard DOT")
		}
		lines = append(lines, "</blockquote>")
	}

	serviceNames := []string{"wg", "wg1", "xr", "np", "oc", "hy", "tg", "ad", "ss", "dnstt", "wp"}
	serviceLabels := []string{
		wireGuardTitle(ctx, b.repo, pac, "wg"),
		wireGuardTitle(ctx, b.repo, pac, "wg1"),
		i18n(pac, "xray"),
		i18n(pac, "naive"),
		i18n(pac, "ocserv"),
		"Hysteria",
		i18n(pac, "mtproto"),
		i18n(pac, "ad_title"),
		i18n(pac, "sh_title"),
		"DNSTT",
		i18n(pac, "warp"),
	}
	serviceColumn := make([]string, 0, len(serviceNames))
	portColumn := make([]string, 0, len(serviceNames))
	for i, name := range serviceNames {
		serviceColumn = append(serviceColumn, dot(serviceAvailable(serviceByName, name))+" "+serviceLabels[i])
		portColumn = append(portColumn, portStatus(serviceByName, name))
	}

	backup := strings.TrimSpace(stringValue(pac["backup"]))
	denyCount := valueCount(pac["deny"])
	autoblock := dot(boolValue(pac["autodeny"])) + " autoblock"
	if denyCount > 0 {
		autoblock += fmt.Sprintf(": %d", denyCount)
	}
	extra := alignColumns([][]string{
		{
			dot(backup != "") + " autobackup",
			dot(boolValue(pac["autoupdate"])) + " autoupdate",
			dot(boolValue(pac["autoscan"])) + " autoscan",
		},
		{
			autoblock,
			dot(boolValue(pac["reset_monthly"])) + " autoreset",
			dot(serviceAvailable(serviceByName, "service")) + " cron",
		},
	})

	lines = append(lines, "", "<code>", alignColumns([][]string{serviceColumn, portColumn}), "", extra, "</code>")
	return strings.Join(lines, "\n")
}

func (b *Bot) legacyPAC(ctx context.Context) map[string]any {
	setting, ok, err := b.repo.GetSetting(ctx, "legacy.pac")
	if err != nil || !ok || setting.ValueJSON == "" {
		return map[string]any{}
	}
	values := map[string]any{}
	if err := json.Unmarshal([]byte(setting.ValueJSON), &values); err != nil {
		return map[string]any{}
	}
	return values
}

func upstreamLikeMainKeyboard(available map[string]bool, pac map[string]any) *telegram.InlineKeyboard {
	rows := [][]telegram.InlineButton{}
	addRow := func(buttons ...telegram.InlineButton) {
		row := make([]telegram.InlineButton, 0, len(buttons))
		for _, button := range buttons {
			if button.Text != "" {
				row = append(row, button)
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	serviceButton := func(service, text, data string) telegram.InlineButton {
		if !available[service] {
			return telegram.InlineButton{}
		}
		return telegram.InlineButton{Text: text, Data: data}
	}
	addRow(
		serviceButton("wg", wireGuardButtonTitle(pac, "wg"), "service:wg"),
		serviceButton("wg1", wireGuardButtonTitle(pac, "wg1"), "service:wg1"),
	)
	addRow(
		serviceButton("xr", i18n(pac, "xray"), "service:xr"),
		serviceButton("np", i18n(pac, "naive"), "service:np"),
	)
	addRow(
		serviceButton("oc", i18n(pac, "ocserv"), "service:oc"),
		serviceButton("tg", i18n(pac, "mtproto"), "service:tg"),
	)
	addRow(
		serviceButton("ad", i18n(pac, "ad_title"), "service:ad"),
		serviceButton("wp", i18n(pac, "warp"), "service:wp"),
	)
	addRow(
		serviceButton("ss", i18n(pac, "sh_title"), "service:ss"),
		serviceButton("xr", i18n(pac, "pac"), "service:pac"),
	)
	dnsttButton := telegram.InlineButton{}
	if boolValue(pac["showdnstt"]) {
		dnsttButton = serviceButton("dnstt", "DNSTT", "service:dnstt")
	}
	addRow(serviceButton("hy", "Hysteria", "service:hy"), dnsttButton)
	addRow(telegram.InlineButton{Text: i18n(pac, "config"), Data: "service:config"})
	domain := stringValue(pac["domain"])
	if domain == "" {
		domain = envDefault("DOMAIN", envDefault("PUBLIC_IP", os.Getenv("IP")))
	}
	donateURL := ""
	if domain != "" {
		donateURL = "https://" + domain + "/webapp" + stringValue(pac["hashbot"]) + "/donate.html"
	}
	donateButton := telegram.InlineButton{}
	if donateURL != "" {
		donateButton = telegram.InlineButton{Text: i18n(pac, "donate"), WebApp: donateURL}
	}
	addRow(
		telegram.InlineButton{Text: i18n(pac, "chat"), URL: "https://t.me/+4G3-Q4d_vFExODcy"},
		donateButton,
	)
	return &telegram.InlineKeyboard{Rows: rows}
}

func wireGuardTitle(ctx context.Context, repo storage.Repository, pac map[string]any, instance string) string {
	key := "amnezia"
	if instance == "wg1" {
		key = "wg1_amnezia"
	}
	if setting, ok, err := repo.GetSetting(ctx, "wireguard."+instance+".amnezia"); err == nil && ok {
		if boolJSON(setting.ValueJSON) {
			return "Amnezia"
		}
		return "Wireguard"
	}
	if boolValue(pac[key]) {
		return "Amnezia"
	}
	return "Wireguard"
}

func wireGuardButtonTitle(pac map[string]any, instance string) string {
	if instance == "wg1" && boolValue(pac["wg1_amnezia"]) {
		return "Amnezia"
	}
	if instance == "wg" && boolValue(pac["amnezia"]) {
		return "Amnezia"
	}
	return "Wireguard"
}

func servicesByName(services []storage.Service) map[string]storage.Service {
	out := make(map[string]storage.Service, len(services))
	for _, service := range services {
		out[service.Name] = service
	}
	return out
}

func serviceAvailable(services map[string]storage.Service, name string) bool {
	service, ok := services[name]
	return ok && service.Enabled && service.Available
}

func serviceEnabled(services map[string]storage.Service, name string) bool {
	service, ok := services[name]
	return ok && service.Enabled
}

func portStatus(services map[string]storage.Service, name string) string {
	switch name {
	case "wg":
		return dot(serviceEnabled(services, name)) + " " + envDefault("WGPORT", "51820")
	case "wg1":
		return dot(serviceEnabled(services, name)) + " " + envDefault("WG1PORT", "51821")
	case "xr", "np", "oc":
		return dot(serviceEnabled(services, name)) + " 443"
	case "hy":
		port := hysteriaPort()
		if port == "" {
			return dot(false) + " port unavailable"
		}
		return dot(serviceEnabled(services, name)) + " " + port
	case "tg":
		return dot(serviceEnabled(services, name)) + " " + envDefault("TGPORT", "4443")
	case "ad":
		return dot(serviceEnabled(services, name)) + " 853"
	case "ss":
		return dot(serviceEnabled(services, name)) + " " + envDefault("SSPORT", "8388")
	case "dnstt":
		return dot(serviceEnabled(services, name)) + " 53"
	default:
		return ""
	}
}

func alignColumns(columns [][]string) string {
	if len(columns) == 0 || len(columns[0]) == 0 {
		return ""
	}
	widths := make([]int, len(columns))
	for idx, column := range columns {
		for _, cell := range column {
			if size := utf8.RuneCountInString(cell); size > widths[idx] {
				widths[idx] = size
			}
		}
	}
	var rows []string
	for row := 0; row < len(columns[0]); row++ {
		parts := make([]string, 0, len(columns))
		for col, column := range columns {
			cell := ""
			if row < len(column) {
				cell = column[row]
			}
			if col < len(columns)-1 {
				cell += strings.Repeat(" ", widths[col]-utf8.RuneCountInString(cell))
			}
			parts = append(parts, cell)
		}
		rows = append(rows, strings.Join(parts, "  "))
	}
	return strings.Join(rows, "\n")
}

func dot(ok bool) string {
	if ok {
		return statusOn
	}
	return statusOff
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func hysteriaPort() string {
	return envDefault("HYPORT", "")
}

func branchSuffix() string {
	branch := os.Getenv("BRANCH")
	if branch == "" {
		return ""
	}
	return " " + branch
}

func boolJSON(value string) bool {
	var out bool
	if err := json.Unmarshal([]byte(value), &out); err == nil {
		return out
	}
	return boolValue(value)
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case string:
		typed = strings.TrimSpace(strings.ToLower(typed))
		return typed == "1" || typed == "true" || typed == "on" || typed == "yes"
	default:
		return false
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func valueCount(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case map[string]any:
		return len(typed)
	default:
		return 0
	}
}
