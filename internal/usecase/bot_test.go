package usecase

import (
	"context"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
)

type repoStub struct {
	admins   map[int64]bool
	services []storage.Service
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
func (r *repoStub) Service(_ context.Context, name string) (storage.Service, bool, error) {
	for _, service := range r.services {
		if service.Name == name {
			return service, true, nil
		}
	}
	return storage.Service{}, false, nil
}
func (r *repoStub) MenuServices(context.Context) ([]storage.Service, error) { return r.services, nil }
func (r *repoStub) SetSetting(context.Context, storage.Setting) error       { return nil }
func (r *repoStub) GetSetting(context.Context, string) (storage.Setting, bool, error) {
	return storage.Setting{}, false, nil
}
func (r *repoStub) SaveClient(context.Context, storage.Client) error { return nil }
func (r *repoStub) ListClients(context.Context, string) ([]storage.Client, error) {
	return nil, nil
}

func TestMenuOnlyUsesRepositoryServices(t *testing.T) {
	repo := &repoStub{
		admins: map[int64]bool{1: true},
		services: []storage.Service{
			{Name: "wg", DisplayName: "WireGuard 0"},
			{Name: "xr", DisplayName: "Xray"},
		},
	}
	result, err := NewBot(repo).HandleMessage(context.Background(), telegram.Message{
		From: telegram.User{ID: 1},
		Chat: telegram.Chat{ID: 10},
		Text: "/menu",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Keyboard == nil || len(result.Keyboard.Rows) != 1 || len(result.Keyboard.Rows[0]) != 2 {
		t.Fatalf("unexpected keyboard: %+v", result.Keyboard)
	}
}
