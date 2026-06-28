package usecase

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
)

func (b *Bot) settingsMenu(ctx context.Context) (MessageResult, error) {
	pac := b.legacyPAC(ctx)
	text := b.settingsText(pac)
	keyboard := &telegram.InlineKeyboard{}
	domain := stringValue(pac["domain"])
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: domainButtonText(pac, domain), Data: "settings:domain"},
		{Text: "nip.io", Data: "settings:nip"},
	})
	if domain != "" {
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: i18n(pac, "Letsencrypt SSL"), Data: "settings:ssl:letsencrypt"},
			{Text: i18n(pac, "Self SSL"), Data: "settings:ssl:self"},
		})
	}
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: "Ports", Data: "svc:ports"},
		{Text: i18n(pac, "logs"), Data: "mod:menu"},
		{Text: "IP ban", Data: "mod:menu"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: i18n(pac, "container management"), Data: "svc:menu"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: i18n(pac, "lang"), Data: "settings:lang"},
		{Text: fmt.Sprintf("%s: %d", i18n(pac, "page"), intValue(pac["limitpage"], 5)), Data: "settings:page"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: i18n(pac, "export"), Data: "backup:export"},
		{Text: i18n(pac, "import"), Data: "backup:import"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
		Text: i18n(pac, "backup") + ": " + backupLabel(pac),
		Data: "settings:backup",
	}})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
		Text: "autoupdate: " + i18n(pac, boolKey(boolValue(pac["autoupdate"]))),
		Data: "settings:autoupdate",
	}})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: "branches", Data: "settings:update"},
		{Text: i18n(pac, "restart"), Data: "settings:restart"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
		Text: i18n(pac, "add") + " " + i18n(pac, "admin"),
		Data: "settings:addadmin",
	}})
	admins, err := b.repo.ListAdmins(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	for _, admin := range admins {
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
			Text: i18n(pac, "delete") + " " + fmt.Sprint(admin.TelegramID),
			Data: fmt.Sprintf("settings:deladmin:%d", admin.TelegramID),
		}})
	}
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
		Text: i18n(pac, "back"),
		Data: "service:menu",
	}})
	return MessageResult{Text: text, Keyboard: keyboard}, nil
}

func (b *Bot) settingsText(pac map[string]any) string {
	domain := stringValue(pac["domain"])
	if domain == "" {
		return i18n(pac, "domain explain")
	}
	lines := []string{"<blockquote>", "Domains:", html.EscapeString(domain)}
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
	return strings.Join(lines, "\n")
}

func (b *Bot) handleSettingsCallback(ctx context.Context, data string) (CallbackResult, bool, error) {
	if !strings.HasPrefix(data, "settings:") {
		return CallbackResult{}, false, nil
	}
	switch data {
	case "settings:menu":
		msg, err := b.settingsMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	default:
		return CallbackResult{Text: "Route not migrated yet", ShowAlert: true}, true, nil
	}
}

func domainButtonText(pac map[string]any, domain string) string {
	if domain == "" {
		return i18n(pac, "install domain")
	}
	return i18n(pac, "delete") + " " + domain
}

func backupLabel(pac map[string]any) string {
	value := strings.TrimSpace(stringValue(pac["backup"]))
	if value == "" {
		return i18n(pac, "off")
	}
	parts := strings.Split(value, "/")
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]) + " start / " + strings.TrimSpace(parts[1]) + " period"
	}
	return i18n(pac, "off") + " " + value + " - wrong format"
}

func boolKey(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		if typed != 0 {
			return typed
		}
	case float64:
		if typed != 0 {
			return int(typed)
		}
	case string:
		var out int
		if _, err := fmt.Sscanf(typed, "%d", &out); err == nil && out != 0 {
			return out
		}
	}
	return fallback
}
