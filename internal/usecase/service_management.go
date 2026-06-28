package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type ServiceController interface {
	SetServiceRunning(ctx context.Context, name string, running bool) error
}

var manageableServices = map[string]string{
	"wg":    "Wireguard",
	"wg1":   "Wireguard 1",
	"xr":    "Vless",
	"np":    "NaiveProxy",
	"oc":    "OpenConnect",
	"tg":    "MTProto",
	"ad":    "AdGuard",
	"wp":    "Warp",
	"ss":    "Shadowsocks",
	"dnstt": "DNSTT",
	"hy":    "Hysteria",
}

var manageableServiceOrder = []string{"wg", "wg1", "xr", "np", "oc", "tg", "ad", "wp", "ss", "dnstt", "hy"}

func (b *Bot) handleServiceManagementCallback(ctx context.Context, data string) (CallbackResult, bool, error) {
	if data == "svc:menu" {
		msg, err := b.serviceManagementMenu(ctx)
		return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
	}
	name, ok := strings.CutPrefix(data, "svc:toggle:")
	if !ok {
		return CallbackResult{}, false, nil
	}
	if _, ok := manageableServices[name]; !ok {
		return CallbackResult{Text: "service is not manageable", ShowAlert: true}, true, nil
	}
	if b.serviceControl == nil {
		return CallbackResult{Text: "service controller unavailable", ShowAlert: true}, true, nil
	}
	service, found, err := b.repo.Service(ctx, name)
	if err != nil {
		return CallbackResult{}, true, err
	}
	if !found {
		return CallbackResult{Text: "service not found", ShowAlert: true}, true, nil
	}
	if !service.Enabled && !service.Available && service.AvailabilityReason != "disabled in bot settings" {
		return CallbackResult{Text: "service disabled in compose", ShowAlert: true}, true, nil
	}
	target := !service.Available
	if err := b.repo.SetSetting(ctx, storage.Setting{
		Key:       "service.disabled." + name,
		ValueJSON: fmt.Sprintf("%t", !target),
	}); err != nil {
		return CallbackResult{}, true, err
	}
	if err := b.serviceControl.SetServiceRunning(ctx, name, target); err != nil {
		return CallbackResult{}, true, err
	}
	service.Available = target
	service.Enabled = target
	service.UpdatedAt = time.Now().UTC()
	if target {
		service.AvailabilityReason = ""
	} else {
		service.AvailabilityReason = "disabled in bot settings"
	}
	if err := b.repo.UpsertService(ctx, service); err != nil {
		return CallbackResult{}, true, err
	}
	msg, err := b.serviceManagementMenu(ctx)
	return CallbackResult{Text: msg.Text, Keyboard: msg.Keyboard}, true, err
}

func (b *Bot) serviceManagementMenu(ctx context.Context) (MessageResult, error) {
	services, err := b.repo.ListServices(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	byName := servicesByName(services)
	lines := []string{"Settings -> Containers", ""}
	keyboard := NewMenuBuilder(1)
	for _, name := range orderedManageableServices(byName) {
		service := byName[name]
		label := manageableServices[name]
		status := dot(service.Enabled && service.Available)
		reason := strings.TrimSpace(service.AvailabilityReason)
		if reason != "" && (!service.Enabled || !service.Available) {
			lines = append(lines, fmt.Sprintf("%s %s: %s", status, label, reason))
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", status, label))
		}
		action := "stop"
		if !service.Available {
			action = "start"
		}
		keyboard.Add(status+" "+action+" "+label, "svc:toggle:"+name)
	}
	if len(lines) == 2 {
		lines = append(lines, "no manageable services")
	}
	keyboard.Add("Back", "service:config")
	return MessageResult{Text: strings.Join(lines, "\n"), Keyboard: keyboard.Build()}, nil
}

func orderedManageableServices(services map[string]storage.Service) []string {
	names := make([]string, 0, len(manageableServiceOrder))
	seen := map[string]bool{}
	for _, name := range manageableServiceOrder {
		if _, ok := services[name]; ok {
			names = append(names, name)
			seen[name] = true
		}
	}
	var extras []string
	for name := range services {
		if _, ok := manageableServices[name]; ok && !seen[name] {
			extras = append(extras, name)
		}
	}
	sort.Strings(extras)
	return append(names, extras...)
}
