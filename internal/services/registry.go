package services

import (
	"context"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type Definition struct {
	Name        string
	DisplayName string
	MenuGroup   string
	SortOrder   int
}

var Definitions = []Definition{
	{Name: "wg", DisplayName: "WireGuard 0", MenuGroup: "main", SortOrder: 10},
	{Name: "wg1", DisplayName: "WireGuard 1", MenuGroup: "main", SortOrder: 20},
	{Name: "xr", DisplayName: "Xray", MenuGroup: "main", SortOrder: 30},
	{Name: "np", DisplayName: "NaiveProxy", MenuGroup: "main", SortOrder: 40},
	{Name: "oc", DisplayName: "OpenConnect", MenuGroup: "main", SortOrder: 50},
	{Name: "tg", DisplayName: "MTProto", MenuGroup: "main", SortOrder: 60},
	{Name: "ad", DisplayName: "AdGuard", MenuGroup: "main", SortOrder: 70},
	{Name: "wp", DisplayName: "Warp", MenuGroup: "main", SortOrder: 80},
	{Name: "ss", DisplayName: "Shadowsocks", MenuGroup: "main", SortOrder: 90},
	{Name: "proxy", DisplayName: "SS Local Proxy", MenuGroup: "support", SortOrder: 100},
	{Name: "dnstt", DisplayName: "DNSTT", MenuGroup: "main", SortOrder: 110},
	{Name: "hy", DisplayName: "Hysteria", MenuGroup: "main", SortOrder: 120},
	{Name: "php", DisplayName: "PHP Legacy Bot", MenuGroup: "support", SortOrder: 900},
	{Name: "service", DisplayName: "Legacy Cron", MenuGroup: "support", SortOrder: 910},
	{Name: "ng", DisplayName: "Nginx", MenuGroup: "support", SortOrder: 920},
	{Name: "up", DisplayName: "Upstream Nginx", MenuGroup: "support", SortOrder: 930},
}

type ComposeReader interface {
	EnabledServices(ctx context.Context) (map[string]bool, error)
}

type RuntimeReader interface {
	RunningServices(ctx context.Context) (map[string]bool, error)
}

type RuntimeController interface {
	SetServiceRunning(ctx context.Context, name string, running bool) error
}

type Registry struct {
	repo    storage.Repository
	compose ComposeReader
	runtime RuntimeReader
}

func NewRegistry(repo storage.Repository, compose ComposeReader, runtime RuntimeReader) *Registry {
	return &Registry{repo: repo, compose: compose, runtime: runtime}
}

func (r *Registry) Refresh(ctx context.Context) error {
	enabled, err := r.compose.EnabledServices(ctx)
	if err != nil {
		return err
	}
	running := map[string]bool{}
	runtimeAvailable := false
	if r.runtime != nil {
		if values, err := r.runtime.RunningServices(ctx); err == nil {
			running = values
			runtimeAvailable = true
		}
	}
	for _, def := range Definitions {
		userDisabled, err := r.userDisabled(ctx, def.Name)
		if err != nil {
			return err
		}
		service := storage.Service{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Enabled:     enabled[def.Name] && !userDisabled,
			Available:   enabled[def.Name] && !userDisabled,
			MenuGroup:   def.MenuGroup,
			SortOrder:   def.SortOrder,
			UpdatedAt:   time.Now().UTC(),
		}
		if userDisabled {
			service.AvailabilityReason = "disabled in bot settings"
			if controller, ok := r.runtime.(RuntimeController); ok {
				_ = controller.SetServiceRunning(ctx, def.Name, false)
			}
		} else if !service.Enabled {
			service.AvailabilityReason = "service disabled in compose"
		} else if runtimeAvailable {
			service.Available = running[def.Name]
			if !service.Available {
				service.AvailabilityReason = "container not running"
			}
		}
		if err := r.repo.UpsertService(ctx, service); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) userDisabled(ctx context.Context, name string) (bool, error) {
	setting, ok, err := r.repo.GetSetting(ctx, "service.disabled."+name)
	if err != nil || !ok {
		return false, err
	}
	return setting.ValueJSON == "true", nil
}
