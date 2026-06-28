package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
	"github.com/ang3el7z/kkk-go-bot/internal/wireguard"
	"github.com/ang3el7z/kkk-go-bot/internal/xray"
)

type repoStub struct {
	admins   map[int64]bool
	services []storage.Service
	settings map[string]storage.Setting
	pending  *storage.PendingOperation
}

func (r *repoStub) Migrate(context.Context) error { return nil }
func (r *repoStub) Close() error                  { return nil }
func (r *repoStub) AddAdmin(_ context.Context, admin storage.Admin) error {
	if r.admins == nil {
		r.admins = map[int64]bool{}
	}
	r.admins[admin.TelegramID] = true
	return nil
}
func (r *repoStub) HasAdmins(context.Context) (bool, error) { return len(r.admins) > 0, nil }
func (r *repoStub) IsAdmin(_ context.Context, id int64) (bool, error) {
	return r.admins[id], nil
}
func (r *repoStub) ListAdmins(context.Context) ([]storage.Admin, error)  { return nil, nil }
func (r *repoStub) UpsertService(context.Context, storage.Service) error { return nil }
func (r *repoStub) ListServices(context.Context) ([]storage.Service, error) {
	return r.services, nil
}
func (r *repoStub) Service(_ context.Context, name string) (storage.Service, bool, error) {
	for _, service := range r.services {
		if service.Name == name {
			return service, true, nil
		}
	}
	return storage.Service{}, false, nil
}
func (r *repoStub) MenuServices(context.Context) ([]storage.Service, error) { return r.services, nil }
func (r *repoStub) SetSetting(_ context.Context, setting storage.Setting) error {
	if r.settings == nil {
		r.settings = map[string]storage.Setting{}
	}
	r.settings[setting.Key] = setting
	return nil
}
func (r *repoStub) GetSetting(_ context.Context, key string) (storage.Setting, bool, error) {
	setting, ok := r.settings[key]
	return setting, ok, nil
}
func (r *repoStub) ListSettings(context.Context, bool) ([]storage.Setting, error) {
	values := make([]storage.Setting, 0, len(r.settings))
	for _, setting := range r.settings {
		values = append(values, setting)
	}
	return values, nil
}
func (r *repoStub) SaveClient(context.Context, storage.Client) error { return nil }
func (r *repoStub) ListClients(context.Context, string) ([]storage.Client, error) {
	return nil, nil
}
func (r *repoStub) DeleteClient(context.Context, string) error { return nil }
func (r *repoStub) SaveWireGuardServer(context.Context, storage.WireGuardServer) error {
	return nil
}
func (r *repoStub) GetWireGuardServer(context.Context, string) (storage.WireGuardServer, bool, error) {
	return storage.WireGuardServer{}, false, nil
}
func (r *repoStub) ListWireGuardServers(context.Context) ([]storage.WireGuardServer, error) {
	return nil, nil
}
func (r *repoStub) SetPendingOperation(_ context.Context, op storage.PendingOperation) error {
	r.pending = &op
	return nil
}
func (r *repoStub) GetPendingOperation(_ context.Context, telegramID int64) (storage.PendingOperation, bool, error) {
	if r.pending == nil || r.pending.TelegramID != telegramID {
		return storage.PendingOperation{}, false, nil
	}
	return *r.pending, true, nil
}
func (r *repoStub) ClearPendingOperation(context.Context, int64) error {
	r.pending = nil
	return nil
}

func TestMenuOnlyUsesRepositoryServices(t *testing.T) {
	repo := &repoStub{
		admins: map[int64]bool{1: true},
		services: []storage.Service{
			{Name: "wg", DisplayName: "WireGuard 0"},
			{Name: "xr", DisplayName: "Xray"},
		},
	}
	result, err := NewBot(repo, wireguard.NewManager(config.Config{}, repo), xray.NewManager(config.Config{}, repo)).HandleMessage(context.Background(), telegram.Message{
		From: telegram.User{ID: 1},
		Chat: telegram.Chat{ID: 10},
		Text: "/menu",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Keyboard == nil || len(result.Keyboard.Rows) != 5 {
		t.Fatalf("unexpected keyboard: %+v", result.Keyboard)
	}
	if result.Keyboard.Rows[0][0].Data != "service:wg" || result.Keyboard.Rows[1][0].Data != "service:xr" {
		t.Fatalf("unexpected go-native callbacks: %+v", result.Keyboard)
	}
}

func TestUnavailableServiceBlocksDirectCallbacks(t *testing.T) {
	repo := &repoStub{
		admins: map[int64]bool{1: true},
		services: []storage.Service{
			{Name: "wg", DisplayName: "WireGuard 0", Enabled: true, Available: false, AvailabilityReason: "container not running"},
			{Name: "xr", DisplayName: "Xray", Enabled: false, Available: false, AvailabilityReason: "service disabled in compose"},
		},
	}
	bot := NewBot(repo, wireguard.NewManager(config.Config{}, repo), xray.NewManager(config.Config{}, repo))
	wgResult, err := bot.HandleCallback(context.Background(), telegram.CallbackQuery{From: telegram.User{ID: 1}, Data: "wg:add:wg"})
	if err != nil {
		t.Fatal(err)
	}
	if !wgResult.ShowAlert || wgResult.Text != "container not running" {
		t.Fatalf("bad wg unavailable result: %+v", wgResult)
	}
	xrayResult, err := bot.HandleCallback(context.Background(), telegram.CallbackQuery{From: telegram.User{ID: 1}, Data: "xray:add:alice"})
	if err != nil {
		t.Fatal(err)
	}
	if !xrayResult.ShowAlert || xrayResult.Text != "service disabled in compose" {
		t.Fatalf("bad xray unavailable result: %+v", xrayResult)
	}
}

func TestUnavailableServiceBlocksMessageRoute(t *testing.T) {
	repo := &repoStub{
		admins: map[int64]bool{1: true},
		services: []storage.Service{
			{Name: "wg", DisplayName: "WireGuard 0", Enabled: true, Available: false, AvailabilityReason: "container not running"},
		},
	}
	result, err := NewBot(repo, wireguard.NewManager(config.Config{}, repo), xray.NewManager(config.Config{}, repo)).HandleMessage(context.Background(), telegram.Message{
		From: telegram.User{ID: 1},
		Chat: telegram.Chat{ID: 10},
		Text: "/wg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "container not running" {
		t.Fatalf("bad unavailable message result: %+v", result)
	}
}

func TestSmallServiceMenuUsesImportedStateAndRedactsSecrets(t *testing.T) {
	repo := &repoStub{
		admins: map[int64]bool{1: true},
		services: []storage.Service{
			{Name: "tg", DisplayName: "MTProto", Enabled: true, Available: true},
		},
		settings: map[string]storage.Setting{
			"legacy.mtprotodomain": {Key: "legacy.mtprotodomain", ValueJSON: `"tg.example"`},
			"legacy.mtprotosecret": {Key: "legacy.mtprotosecret", ValueJSON: `"secret"`, Secret: true},
		},
	}
	result, err := NewBot(repo, wireguard.NewManager(config.Config{}, repo), xray.NewManager(config.Config{}, repo)).HandleCallback(context.Background(), telegram.CallbackQuery{
		From: telegram.User{ID: 1},
		Data: "service:tg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Text, "MTProto") || !strings.Contains(result.Text, "tg.example") || !strings.Contains(result.Text, "secret present") || strings.Contains(result.Text, `"secret"`) {
		t.Fatalf("bad small service menu: %+v", result)
	}
	if result.Keyboard == nil || len(result.Keyboard.Rows) != 1 || result.Keyboard.Rows[0][0].Data != "service:menu" {
		t.Fatalf("missing back keyboard: %+v", result.Keyboard)
	}
}
