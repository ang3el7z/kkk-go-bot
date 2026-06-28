package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/adguard"
	"github.com/ang3el7z/kkk-go-bot/internal/moderation"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
	"github.com/ang3el7z/kkk-go-bot/internal/wireguard"
	"github.com/ang3el7z/kkk-go-bot/internal/xray"
)

type Bot struct {
	repo storage.Repository
	wg   *wireguard.Manager
	xray *xray.Manager
	ad   *adguard.Manager
	mod  *moderation.Manager
}

func NewBot(repo storage.Repository, wg *wireguard.Manager, xr *xray.Manager, extras ...any) *Bot {
	bot := &Bot{repo: repo, wg: wg, xray: xr}
	for _, extra := range extras {
		switch typed := extra.(type) {
		case *adguard.Manager:
			bot.ad = typed
		case *moderation.Manager:
			bot.mod = typed
		}
	}
	return bot
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
		if err := b.requireServiceAvailable(ctx, "wg"); err != nil {
			return MessageResult{Text: err.Error()}, nil
		}
		return b.wgMenu(ctx, "wg")
	case "/xray":
		if err := b.requireServiceAvailable(ctx, "xr"); err != nil {
			return MessageResult{Text: err.Error()}, nil
		}
		return b.xrayMenu(ctx)
	case "/logs", "/deny", "/ip":
		return b.moderationMenu(ctx)
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
	if result, ok, err := b.handleAdGuardCallback(ctx, query.From.ID, query.Data); ok || err != nil {
		return result, err
	}
	if result, ok, err := b.handleModerationCallback(ctx, query.From.ID, query.Data); ok || err != nil {
		return result, err
	}
	name, ok := strings.CutPrefix(query.Data, "service:")
	if !ok {
		if query.Data == "/menu wg" || strings.HasPrefix(query.Data, "/menu wg ") || query.Data == "/changeWG 0" {
			if err := b.requireServiceAvailable(ctx, "wg"); err != nil {
				return CallbackResult{Text: err.Error(), ShowAlert: true}, nil
			}
			msg, err := b.wgMenu(ctx, "wg")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		if query.Data == "/changeWG 1" {
			if err := b.requireServiceAvailable(ctx, "wg1"); err != nil {
				return CallbackResult{Text: err.Error(), ShowAlert: true}, nil
			}
			msg, err := b.wgMenu(ctx, "wg1")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		if query.Data == "/add" {
			if err := b.requireServiceAvailable(ctx, "wg"); err != nil {
				return CallbackResult{Text: err.Error(), ShowAlert: true}, nil
			}
			_, _, err := b.wg.Add(ctx, "wg", "all", "")
			if err != nil {
				return CallbackResult{}, err
			}
			msg, err := b.wgMenu(ctx, "wg")
			return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
		}
		return CallbackResult{Text: "Route not migrated yet", ShowAlert: true}, nil
	}
	if name == "menu" {
		msg, err := b.menu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
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
	if result, ok, err := b.smallServiceMenu(ctx, name); ok || err != nil {
		return result, err
	}
	if name == "ad" {
		msg, err := b.adguardMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, err
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
	if err := b.requireServiceAvailable(ctx, wireGuardServiceName(value)); err != nil {
		return CallbackResult{Text: err.Error(), ShowAlert: true}, true, nil
	}
	switch action {
	case "add":
		client, _, err := b.wg.Add(ctx, value, "all", "")
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
	case "amnezia":
		enabled, err := b.wg.ToggleAmnezia(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.wgMenu(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg.Text = fmt.Sprintf("Amnezia: %t\n\n%s", enabled, msg.Text)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, nil
	case "endpoint":
		enabled, err := b.wg.ToggleEndpoint(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.wgMenu(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg.Text = fmt.Sprintf("Endpoint IP: %t\n\n%s", enabled, msg.Text)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, nil
	case "torrent":
		enabled, err := b.wg.ToggleBlockTorrent(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: fmt.Sprintf("Torrent block: %t", enabled), ShowAlert: false}, true, nil
	case "exchange":
		enabled, err := b.wg.ToggleBlockExchange(ctx, value)
		if err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: fmt.Sprintf("Exchange block: %t", enabled), ShowAlert: false}, true, nil
	case "subnetadd":
		if err := b.setPending(ctx, telegramID, action, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send subnet CIDR, e.g. 10.0.0.0/8", ShowAlert: true}, true, nil
	case "subnetdel":
		instance, subnet, ok := strings.Cut(value, ":")
		if !ok {
			return CallbackResult{Text: "Bad subnet delete action", ShowAlert: true}, true, nil
		}
		if err := b.wg.DeleteSubnet(ctx, instance, subnet); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.wgMenu(ctx, instance)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "defaultallowedips":
		if err := b.setPending(ctx, telegramID, action, value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send default AllowedIPs for new peers, e.g. 0.0.0.0/0", ShowAlert: true}, true, nil
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
	case "wg_defaultallowedips":
		err = b.wg.SetDefaultAllowedIPs(ctx, payload.ClientID, msg.Text)
	case "wg_subnetadd":
		err = b.wg.AddSubnet(ctx, payload.ClientID, msg.Text)
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
	case "xray_hwid_default":
		var count int
		if _, scanErr := fmt.Sscanf(msg.Text, "%d", &count); scanErr != nil {
			return MessageResult{}, true, scanErr
		}
		err = b.xray.SetDefaultHWIDLimit(ctx, count)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "xray_hwid_client":
		var count int
		if _, scanErr := fmt.Sscanf(msg.Text, "%d", &count); scanErr != nil {
			return MessageResult{}, true, scanErr
		}
		err = b.xray.SetClientHWIDLimit(ctx, payload.ClientID, count)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "xray_route_add":
		err = b.xray.AddRouteItem(ctx, payload.ClientID, msg.Text)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "xray_template_add":
		typ, _, ok := strings.Cut(payload.ClientID, ":")
		if !ok {
			return MessageResult{Text: "Bad template payload"}, true, nil
		}
		name, templateBody, ok := strings.Cut(msg.Text, "\n")
		if !ok {
			return MessageResult{Text: "Send template as: name newline JSON body"}, true, nil
		}
		err = b.xray.AddTemplate(ctx, typ, name, templateBody)
		if err == nil {
			result, err := b.xrayMenu(ctx)
			return result, true, err
		}
	case "adguard_upstream_add":
		err = b.ad.AddUpstream(ctx, msg.Text)
		if err == nil {
			result, err := b.adguardMenu(ctx)
			return result, true, err
		}
	case "moderation_deny_add":
		err = b.mod.AddDeny(ctx, msg.Text)
		if err == nil {
			result, err := b.moderationMenu(ctx)
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
	if err := b.requireServiceAvailable(ctx, "xr"); err != nil {
		return CallbackResult{Text: err.Error(), ShowAlert: true}, true, nil
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
	case "resetuuid":
		if err := b.xray.ResetUUID(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "transport":
		if err := b.xray.SetTransport(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "hwidglobal":
		enabled, err := b.xray.ToggleGlobalHWID(ctx)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		if err != nil {
			return CallbackResult{}, true, err
		}
		msg.Text = fmt.Sprintf("HWID global: %t\n\n%s", enabled, msg.Text)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, nil
	case "hwiddefault":
		if err := b.setPendingOperation(ctx, telegramID, "xray_hwid_default", ""); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send default HWID device count", ShowAlert: true}, true, nil
	case "hwidtoggle":
		if err := b.xray.ToggleClientHWID(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "hwidlimit":
		if err := b.setPendingOperation(ctx, telegramID, "xray_hwid_client", value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send per-user HWID device count, or 0 for default", ShowAlert: true}, true, nil
	case "routeadd":
		if err := b.setPendingOperation(ctx, telegramID, "xray_route_add", value); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send route value for " + value, ShowAlert: true}, true, nil
	case "routedel":
		list, item, ok := strings.Cut(value, ":")
		if !ok {
			return CallbackResult{Text: "Bad route delete action", ShowAlert: true}, true, nil
		}
		if err := b.xray.DeleteRouteItem(ctx, list, item); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "templateadd":
		if err := b.setPendingOperation(ctx, telegramID, "xray_template_add", value+":template"); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send template as: name newline JSON body", ShowAlert: true}, true, nil
	case "templatedel":
		typ, name, ok := strings.Cut(value, ":")
		if !ok {
			return CallbackResult{Text: "Bad template delete action", ShowAlert: true}, true, nil
		}
		if err := b.xray.DeleteTemplate(ctx, typ, name); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.xrayMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
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

func (b *Bot) handleAdGuardCallback(ctx context.Context, telegramID int64, data string) (CallbackResult, bool, error) {
	if b.ad == nil || !strings.HasPrefix(data, "ad:") {
		return CallbackResult{}, false, nil
	}
	if err := b.requireServiceAvailable(ctx, "ad"); err != nil {
		return CallbackResult{Text: err.Error(), ShowAlert: true}, true, nil
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return CallbackResult{Text: "Bad AdGuard action", ShowAlert: true}, true, nil
	}
	action := parts[1]
	value := ""
	if len(parts) == 3 {
		value = parts[2]
	}
	switch action {
	case "menu":
		msg, err := b.adguardMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "upstreamadd":
		if err := b.setPendingOperation(ctx, telegramID, "adguard_upstream_add", "ad"); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send upstream DNS, e.g. https://dns.google/dns-query", ShowAlert: true}, true, nil
	case "upstreamdel":
		if err := b.ad.DeleteUpstream(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.adguardMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	default:
		return CallbackResult{Text: "Unknown AdGuard action", ShowAlert: true}, true, nil
	}
}

func (b *Bot) handleModerationCallback(ctx context.Context, telegramID int64, data string) (CallbackResult, bool, error) {
	if b.mod == nil || !strings.HasPrefix(data, "mod:") {
		return CallbackResult{}, false, nil
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return CallbackResult{Text: "Bad moderation action", ShowAlert: true}, true, nil
	}
	action := parts[1]
	value := ""
	if len(parts) == 3 {
		value = parts[2]
	}
	switch action {
	case "menu":
		msg, err := b.moderationMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "denyadd":
		if err := b.setPendingOperation(ctx, telegramID, "moderation_deny_add", "deny"); err != nil {
			return CallbackResult{}, true, err
		}
		return CallbackResult{Text: "Send IP/CIDR to deny, e.g. 203.0.113.10 or 203.0.113.0/24", ShowAlert: true}, true, nil
	case "denydelete":
		if err := b.mod.DeleteDeny(ctx, value); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.moderationMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "denyclear":
		if err := b.mod.ClearDeny(ctx); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.moderationMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "logclear":
		if err := b.mod.ClearLogs(); err != nil {
			return CallbackResult{}, true, err
		}
		msg, err := b.moderationMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	case "logtail":
		body, err := b.mod.TailLog(value, 3500)
		if err != nil {
			return CallbackResult{}, true, err
		}
		if strings.TrimSpace(body) == "" {
			body = "empty log"
		}
		return CallbackResult{Text: body, ShowAlert: true}, true, nil
	default:
		return CallbackResult{Text: "Unknown moderation action", ShowAlert: true}, true, nil
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
	builder := NewMenuBuilder(2)
	for _, service := range services {
		builder.Add(service.DisplayName, callbackForService(service.Name))
	}
	keyboard := builder.Build()
	if keyboard == nil {
		return MessageResult{Text: "No enabled services found in compose."}, nil
	}
	return MessageResult{Text: "kkk-go-bot menu", Keyboard: keyboard}, nil
}

func (b *Bot) wgMenu(ctx context.Context, instance string) (MessageResult, error) {
	info, err := b.wg.Info(ctx, instance)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{Rows: [][]telegram.InlineButton{{
		{Text: "Add peer", Data: "wg:add:" + instance},
		{Text: fmt.Sprintf("Amnezia: %t", info.Amnezia), Data: "wg:amnezia:" + instance},
		{Text: "Default AllowedIPs", Data: "wg:defaultallowedips:" + instance},
	}}}
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: fmt.Sprintf("Endpoint IP: %t", info.EndpointUseIP), Data: "wg:endpoint:" + instance},
		{Text: fmt.Sprintf("Torrent: %t", info.BlockTorrent), Data: "wg:torrent:" + instance},
		{Text: fmt.Sprintf("Exchange: %t", info.BlockExchange), Data: "wg:exchange:" + instance},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: "Add subnet", Data: "wg:subnetadd:" + instance},
	})
	lines := []string{"default allow=" + info.DefaultAllowedIPs}
	if len(info.Subnets) > 0 {
		lines = append(lines, "subnets="+strings.Join(info.Subnets, ","))
		for _, subnet := range info.Subnets {
			keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{
				Text: "delete " + subnet,
				Data: "wg:subnetdel:" + instance + ":" + subnet,
			}})
		}
	}
	for _, client := range info.Clients {
		status := "off"
		if client.Enabled {
			status = "on"
		}
		details := []string{client.Address}
		if client.AllowedIPs != "" {
			details = append(details, "allow="+client.AllowedIPs)
		}
		if client.DNS != "" {
			details = append(details, "dns="+client.DNS)
		}
		if client.MTU != "" {
			details = append(details, "mtu="+client.MTU)
		}
		if client.ExpiresAt != "" {
			details = append(details, "until="+client.ExpiresAt)
		}
		if client.Handshake != "" {
			details = append(details, "handshake="+client.Handshake)
		}
		if client.Transfer != "" {
			details = append(details, "transfer="+client.Transfer)
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", status, client.Name, strings.Join(details, " ")))
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

func (b *Bot) adguardMenu(ctx context.Context) (MessageResult, error) {
	if b.ad == nil {
		return MessageResult{Text: "AdGuard adapter unavailable"}, nil
	}
	info, err := b.ad.Info(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := NewMenuBuilder(1)
	keyboard.Add("Add upstream", "ad:upstreamadd:")
	for _, upstream := range info.Upstreams {
		keyboard.Add("delete "+upstream, "ad:upstreamdel:"+upstream)
	}
	keyboard.Add("Back", "service:menu")
	lines := []string{
		"AdGuard",
		fmt.Sprintf("protection=%t", info.ProtectionEnabled),
		fmt.Sprintf("bind=%s:%d", info.BindHost, info.BindPort),
		fmt.Sprintf("users=%d", info.Users),
	}
	if len(info.Upstreams) > 0 {
		lines = append(lines, "upstreams="+strings.Join(info.Upstreams, ","))
	} else {
		lines = append(lines, "upstreams=none")
	}
	return MessageResult{Text: strings.Join(lines, "\n"), Keyboard: keyboard.Build()}, nil
}

func (b *Bot) moderationMenu(ctx context.Context) (MessageResult, error) {
	if b.mod == nil {
		return MessageResult{Text: "Moderation adapter unavailable"}, nil
	}
	info, err := b.mod.Info()
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := NewMenuBuilder(1)
	keyboard.Add("Add deny IP/CIDR", "mod:denyadd:")
	if len(info.Deny) > 0 {
		keyboard.Add("Clear deny list", "mod:denyclear:")
	}
	for _, value := range info.Deny {
		keyboard.Add("allow "+value, "mod:denydelete:"+value)
	}
	if len(info.Logs) > 0 {
		keyboard.Add("Clear logs", "mod:logclear:")
	}
	for _, logFile := range info.Logs {
		keyboard.Add("tail "+logFile.Name, "mod:logtail:"+logFile.Name)
	}
	lines := []string{"Moderation", fmt.Sprintf("deny=%d", len(info.Deny)), fmt.Sprintf("logs=%d", len(info.Logs))}
	for _, logFile := range info.Logs {
		lines = append(lines, fmt.Sprintf("log %s %dB", logFile.Name, logFile.Size))
	}
	return MessageResult{Text: strings.Join(lines, "\n"), Keyboard: keyboard.Build()}, nil
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
	info, err := b.xray.Info(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{Rows: [][]telegram.InlineButton{{{Text: "Add user", Data: "xray:add"}}}}
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: "WS", Data: "xray:transport:Websocket"},
		{Text: "Reality", Data: "xray:transport:Reality"},
		{Text: "XHTTP", Data: "xray:transport:xhttp"},
	})
	keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
		{Text: fmt.Sprintf("HWID: %t", info.HWIDEnabled), Data: "xray:hwidglobal"},
		{Text: fmt.Sprintf("Default HWID: %d", info.HWIDDefault), Data: "xray:hwiddefault"},
	})
	for _, list := range []string{"block", "warp", "proxy", "subnet", "process", "package", "ruleset"} {
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{Text: "Add route " + list, Data: "xray:routeadd:" + list}})
	}
	for list, values := range routeValues(info.Routes) {
		for _, value := range values {
			keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{Text: "del " + list + " " + value, Data: "xray:routedel:" + list + ":" + value}})
		}
	}
	for _, typ := range []string{"v2ray", "sing", "clash"} {
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{Text: "Add template " + typ, Data: "xray:templateadd:" + typ}})
	}
	for typ, values := range templateValues(info.Templates) {
		for _, value := range values {
			keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{{Text: "del template " + typ + " " + value, Data: "xray:templatedel:" + typ + ":" + value}})
		}
	}
	lines := []string{"transport=" + info.Transport}
	lines = append(lines, routeSummary(info.Routes)...)
	lines = append(lines, "templates v2ray="+strings.Join(info.Templates.V2Ray, ","), "templates sing="+strings.Join(info.Templates.Sing, ","), "templates clash="+strings.Join(info.Templates.Clash, ","))
	for _, client := range info.Clients {
		status := "off"
		if client.Enabled {
			status = "on"
		}
		traffic := ""
		if client.Download != "" || client.Upload != "" {
			traffic = fmt.Sprintf(" down=%s up=%s", client.Download, client.Upload)
		}
		hwid := ""
		if info.HWIDEnabled {
			limit := info.HWIDDefault
			if client.HWIDLimit > 0 {
				limit = client.HWIDLimit
			}
			hwid = fmt.Sprintf(" hwid=%d", limit)
			if client.HWIDDisabled {
				hwid = " hwid=off"
			}
		}
		lines = append(lines, fmt.Sprintf("%s %s%s%s", status, client.Name, traffic, hwid))
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "toggle " + client.Name, Data: "xray:toggle:" + client.ID},
			{Text: "link", Data: "xray:link:" + client.ID},
			{Text: "QR", Data: "xray:qr:" + client.ID},
		})
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "rename", Data: "xray:rename:" + client.ID},
			{Text: "timer", Data: "xray:timer:" + client.ID},
			{Text: "reset stats", Data: "xray:resetstats:" + client.ID},
			{Text: "reset uuid", Data: "xray:resetuuid:" + client.ID},
			{Text: "delete", Data: "xray:delete:" + client.ID},
		})
		keyboard.Rows = append(keyboard.Rows, []telegram.InlineButton{
			{Text: "HWID on/off", Data: "xray:hwidtoggle:" + client.ID},
			{Text: "HWID limit", Data: "xray:hwidlimit:" + client.ID},
		})
	}
	text := "Xray"
	if len(lines) > 0 {
		text += "\n\n" + strings.Join(lines, "\n")
	}
	return MessageResult{Text: text, Keyboard: keyboard}, nil
}

func routeSummary(routes xray.RouteLists) []string {
	return []string{
		"routes block=" + strings.Join(routes.Block, ","),
		"routes warp=" + strings.Join(routes.Warp, ","),
		"routes proxy=" + strings.Join(routes.Proxy, ","),
		"routes subnet=" + strings.Join(routes.Subnet, ","),
		"routes process=" + strings.Join(routes.Process, ","),
		"routes package=" + strings.Join(routes.Package, ","),
		"routes ruleset=" + strings.Join(routes.RuleSets, ","),
	}
}

func routeValues(routes xray.RouteLists) map[string][]string {
	return map[string][]string{
		"block":   routes.Block,
		"warp":    routes.Warp,
		"proxy":   routes.Proxy,
		"subnet":  routes.Subnet,
		"process": routes.Process,
		"package": routes.Package,
		"ruleset": routes.RuleSets,
	}
}

func templateValues(templates xray.TemplateInfo) map[string][]string {
	return map[string][]string{
		"v2ray": templates.V2Ray,
		"sing":  templates.Sing,
		"clash": templates.Clash,
	}
}

func (b *Bot) requireServiceAvailable(ctx context.Context, name string) error {
	service, found, err := b.repo.Service(ctx, name)
	if err != nil {
		return err
	}
	if !found || !service.Enabled || !service.Available {
		reason := service.AvailabilityReason
		if reason == "" {
			reason = "service unavailable"
		}
		return fmt.Errorf("%s", reason)
	}
	return nil
}

func wireGuardServiceName(value string) string {
	service := value
	if before, _, ok := strings.Cut(value, ":"); ok {
		service = before
	}
	if strings.HasPrefix(service, "wg1") {
		return "wg1"
	}
	return "wg"
}
