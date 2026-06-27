package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
	"github.com/ang3el7z/kkk-go-bot/internal/wireguard"
	"github.com/ang3el7z/kkk-go-bot/internal/xray"
)

type Bot struct {
	repo storage.Repository
	wg   *wireguard.Manager
	xray *xray.Manager
}

func NewBot(repo storage.Repository, wg *wireguard.Manager, xr *xray.Manager) *Bot {
	return &Bot{repo: repo, wg: wg, xray: xr}
}

type MessageResult struct {
	Text     string
	Keyboard *telegram.InlineKeyboard
}

type CallbackResult struct {
	Text      string
	ShowAlert bool
	Keyboard  *telegram.InlineKeyboard
	Document  *telegram.Document
	Photo     *telegram.Photo
}

func (b *Bot) HandleMessage(ctx context.Context, msg telegram.Message) (MessageResult, error) {
	if msg.Text == "/id" {
		return MessageResult{Text: fmt.Sprintf("User ID: %d\nChat ID: %d", msg.From.ID, msg.Chat.ID)}, nil
	}
	if err := b.requireAdminOrBootstrap(ctx, msg.From); err != nil {
		return MessageResult{Text: "Unauthorized"}, nil
	}
	if result, ok, err := b.handlePendingMessage(ctx, msg); ok || err != nil {
		return result, err
	}
	switch msg.Text {
	case "/start", "/menu", "":
		return b.menu(ctx)
	case "/wg", "/wireguard":
		return b.wgMenu(ctx, "wg")
	case "/xray":
		return b.xrayMenu(ctx)
	default:
		return MessageResult{Text: "Unknown command. Use /menu."}, nil
	}
}

func (b *Bot) HandleCallback(ctx context.Context, query telegram.CallbackQuery) (CallbackResult, error) {
	if err := b.requireAdminOrBootstrap(ctx, query.From); err != nil {
		return CallbackResult{Text: "Unauthorized", ShowAlert: true}, nil
	}
	if result, ok, err := b.handleWireGuardCallback(ctx, query.From.ID, query.Data); ok || err != nil {
		return result, err
	}
	if result, ok, err := b.handleXrayCallback(ctx, query.From.ID, query.Data); ok || err != nil {
		return result, err
	}
	name, ok := strings.CutPrefix(query.Data, "service:")
	if !ok {
		if query.Data == "/menu wg" || strings.HasPrefix(query.Data, "/menu wg ") || query.Data == "/changeWG 0" {
			msg, err := b.wgMenu(ctx, "wg")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		if query.Data == "/changeWG 1" {
			msg, err := b.wgMenu(ctx, "wg1")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		if query.Data == "/add" {
			_, _, err := b.wg.Add(ctx, "wg", "all", "0.0.0.0/0")
			if err != nil {
				return CallbackResult{}, err
			}
			msg, err := b.wgMenu(ctx, "wg")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		return CallbackResult{Text: "Route not migrated yet", ShowAlert: true}, nil
	}
	service, found, err := b.repo.Service(ctx, name)
	if err != nil {
		return CallbackResult{}, err
	}
	if !found || !service.Enabled || !service.Available {
		reason := service.AvailabilityReason
		if reason == "" {
			reason = "service unavailable"
		}
		return CallbackResult{Text: reason, ShowAlert: true}, nil
	}
	return CallbackResult{Text: service.DisplayName, ShowAlert: false}, nil
}

func (b *Bot) handleWireGuardCallback(ctx context.Context, telegramID int64, data string) (CallbackResult, bool, error) {
	if b.wg == nil || !strings.HasPrefix(data, "wg:") {
		return CallbackResult{}, false, nil
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return CallbackResult{Text: "Bad WireGuard action", ShowAlert: true}, true, nil
	}
	action := parts[1]
	value := parts[2]
	switch action {
	case "add":
		client, _, err := b.wg.Add(ctx, value, "all", "0.0.0.0/0")
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Created " + client.Name, ShowAlert: false}, true, nil
	case "toggle":
		if err := b.wg.Toggle(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "WireGuard client toggled", ShowAlert: false}, true, nil
	case "delete":
		if err := b.wg.Delete(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "WireGuard client deleted", ShowAlert: false}, true, nil
	case "download":
		filename, conf, err := b.wg.ClientConfig(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Document: &telegram.Document{Filename: filename, Content: []byte(conf)}}, true, nil
	case "qr":
		filename, png, err := b.wg.ClientQR(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Photo: &telegram.Photo{Filename: filename, Content: png, Caption: "WireGuard QR"}}, true, nil
	case "rename", "timer", "dns", "mtu", "allowedips":
		if err := b.setPending(ctx, telegramID, action, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: promptForWGAction(action), ShowAlert: true}, true, nil
	default:
		return CallbackResult{Text: "Unknown WireGuard action", ShowAlert: true}, true, nil
	}
}

func (b *Bot) handlePendingMessage(ctx context.Context, msg telegram.Message) (MessageResult, bool, error) {
	op, ok, err := b.repo.GetPendingOperation(ctx, msg.From.ID)
	if err != nil || !ok {
		return MessageResult{}, ok, err
	}
	if err := b.repo.ClearPendingOperation(ctx, msg.From.ID); err != nil {
		return MessageResult{}, true, err
	}
	var payload struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal([]byte(op.PayloadJSON), &payload); err != nil {
		return MessageResult{}, true, err
	}
	switch op.Operation {
	case "wg_rename":
		err = b.wg.Rename(ctx, payload.ClientID, msg.Text)
	case "wg_timer":
		err = b.wg.SetExpiry(ctx, payload.ClientID, msg.Text)
	case "wg_dns":
		err = b.wg.SetDNS(ctx, payload.ClientID, msg.Text)
	case "wg_mtu":
		err = b.wg.SetMTU(ctx, payload.ClientID, msg.Text)
	case "wg_allowedips":
		err = b.wg.SetAllowedIPs(ctx, payload.ClientID, msg.Text)
	case "xray_add":
		_, err = b.xray.Add(ctx, msg.Text)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "xray_rename":
		err = b.xray.Rename(ctx, payload.ClientID, msg.Text)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "xray_timer":
		err = b.xray.SetTimer(ctx, payload.ClientID, msg.Text)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	default:
		return MessageResult{Text: "Unknown pending operation"}, true, nil
	}
	if err != nil {
		return MessageResult{}, true, err
	}
	instance, _, _ := strings.Cut(payload.ClientID, ":")
	result, err := b.wgMenu(ctx, instance)
	return result, true, err
}

func (b *Bot) handleXrayCallback(ctx context.Context, telegramID int64, data string) (CallbackResult, bool, error) {
	if b.xray == nil || !strings.HasPrefix(data, "xray:") {
		return CallbackResult{}, false, nil
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return CallbackResult{Text: "Bad Xray action", ShowAlert: true}, true, nil
	}
	action := parts[1]
	value := ""
	if len(parts) == 3 {
		value = parts[2]
	}
	switch action {
	case "menu":
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "add":
		if err := b.setPendingOperation(ctx, telegramID, "xray_add", ""); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send Xray user name/email", ShowAlert: true}, true, nil
	case "toggle":
		if err := b.xray.Toggle(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "delete":
		if err := b.xray.Delete(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "rename":
		if err := b.setPendingOperation(ctx, telegramID, "xray_rename", value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send new Xray user name/email", ShowAlert: true}, true, nil
	case "timer":
		if err := b.setPendingOperation(ctx, telegramID, "xray_timer", value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send expiry as YYYY-MM-DD HH:MM:SS, or 0 to clear", ShowAlert: true}, true, nil
	case "resetstats":
		if err := b.xray.ResetUserStats(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Xray stats reset marker saved", ShowAlert: false}, true, nil
	case "link":
		link, err := b.xray.Link(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: link, ShowAlert: true}, true, nil
	case "qr":
		filename, png, err := b.xray.QR(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Photo: &telegram.Photo{Filename: filename, Content: png, Caption: "Xray QR"}}, true, nil
	default:
		return CallbackResult{Text: "Unknown Xray action", ShowAlert: true}, true, nil
	}
}

func (b *Bot) setPending(ctx context.Context, telegramID int64, action, clientID string) error {
	return b.setPendingOperation(ctx, telegramID, "wg_"+action, clientID)
}

func (b *Bot) setPendingOperation(ctx context.Context, telegramID int64, operation, clientID string) error {
	payload, err := json.Marshal(map[string]string{"client_id": clientID})
	if err != nil {
		return err
	}
	return b.repo.SetPendingOperation(ctx, storage.PendingOperation{
		TelegramID:  telegramID,
		Operation:   operation,
		PayloadJSON: string(payload),
		ExpiresAt:   time.Now().UTC().Add(15 * time.Minute),
	})
}

func (b *Bot) requireAdminOrBootstrap(ctx context.Context, user telegram.User) error {
	hasAdmins, err := b.repo.HasAdmins(ctx)
	if err != nil {
		return err
	}
	if !hasAdmins {
		return b.repo.AddAdmin(ctx, storage.Admin{
			TelegramID: user.ID,
			Username:   user.Username,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
		})
	}
	ok, err := b.repo.IsAdmin(ctx, user.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unauthorized")
	}
	return nil
}

func (b *Bot) menu(ctx context.Context) (MessageResult, error) {
	services, err := b.repo.MenuServices(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{}
	for i := 0; i < len(services); i += 2 {
		row := []telegram.InlineButton{{
			Text: services[i].DisplayName,
			Data: callbackForService(services[i].Name),
		}}
		if i+1 < len(services) {
			row = append(row, telegram.InlineButton{
				Text: services[i+1].DisplayName,
				Data: callbackForService(services[i+1].Name),
			})
		}
		keyboard.Rows = append(keyboard.Rows, row)
	}
	if len(keyboard.Rows) == 0 {
		return MessageResult{Text: "No enabled services found in compose."}, nil
	}
	return MessageResult{Text: "kkk-go-bot menu", Keyboard: keyboard}, nil
}

func (b *Bot) wgMenu(ctx context.Context, instance string) (MessageResult, error) {
	clients, err := b.wg.List(ctx, instance)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{Rows: [][]telegram.InlineButton{{
		{Text: "Add peer", Data: "wg:add:" + instance},
	}}}
	var lines []string
	for _, client := range clients {
		status := "off"
		if client.Enabled {
			status = "on"
		}
		lines = append(lines, fmt.Sprintf("%s %s", status, client.Name))
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "toggle " + client.Name, Data: "wg:toggle:" + client.ID},
			{Text: "QR", Data: "wg:qr:" + client.ID},
			{Text: "download", Data: "wg:download:" + client.ID},
			{Text: "delete", Data: "wg:delete:" + client.ID},
		})
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "rename", Data: "wg:rename:" + client.ID},
			{Text: "timer", Data: "wg:timer:" + client.ID},
			{Text: "DNS", Data: "wg:dns:" + client.ID},
			{Text: "MTU", Data: "wg:mtu:" + client.ID},
		})
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "AllowedIPs", Data: "wg:allowedips:" + client.ID},
		})
	}
	text := "WireGuard " + instance
	if len(lines) > 0 {
		text += "\n\n" + strings.Join(lines, "\n")
	}
	return MessageResult{Text: text, Keyboard: keyboard}, nil
}

func promptForWGAction(action string) string {
	switch action {
	case "rename":
		return "Send new client name"
	case "timer":
		return "Send expiry as YYYY-MM-DD HH:MM:SS, or 0 to clear"
	case "dns":
		return "Send DNS servers separated by comma, or 0 to clear"
	case "mtu":
		return "Send MTU, or 0 to clear"
	case "allowedips":
		return "Send AllowedIPs, e.g. 0.0.0.0/0"
	default:
		return "Send value"
	}
}

func callbackForService(name string) string {
	switch name {
	case "wg":
		return "/menu wg"
	case "wg1":
		return "/changeWG 1"
	case "xr":
		return "xray:menu"
	default:
		return "service:" + name
	}
}

func (b *Bot) xrayMenu(ctx context.Context) (MessageResult, error) {
	clients, err := b.xray.List(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{Rows: [][]telegram.InlineButton{{{Text: "Add user", Data: "xray:add"}}}}
	lines := make([]string, 0, len(clients))
	for _, client := range clients {
		status := "off"
		if client.Enabled {
			status = "on"
		}
		lines = append(lines, fmt.Sprintf("%s %s", status, client.Name))
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "toggle " + client.Name, Data: "xray:toggle:" + client.ID},
			{Text: "link", Data: "xray:link:" + client.ID},
			{Text: "QR", Data: "xray:qr:" + client.ID},
		})
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "rename", Data: "xray:rename:" + client.ID},
			{Text: "timer", Data: "xray:timer:" + client.ID},
			{Text: "reset stats", Data: "xray:resetstats:" + client.ID},
			{Text: "delete", Data: "xray:delete:" + client.ID},
		})
	}
	text := "Xray"
	if len(lines) > 0 {
		text += "\n\n" + strings.Join(lines, "\n")
	}
	return MessageResult{Text: text, Keyboard: keyboard}, nil
}
