package services

import (
	"context"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type memoryRepo struct {
	services map[string]storage.Service
}

func (m *memoryRepo) Migrate(context.Context) error { return nil }
func (m *memoryRepo) Close() error                  { return nil }
func (m *memoryRepo) AddAdmin(context.Context, storage.Admin) error {
	return nil
}
func (m *memoryRepo) HasAdmins(context.Context) (bool, error) { return false, nil }
func (m *memoryRepo) IsAdmin(context.Context, int64) (bool, error) {
	return false, nil
}
func (m *memoryRepo) ListAdmins(context.Context) ([]storage.Admin, error) { return nil, nil }
func (m *memoryRepo) UpsertService(_ context.Context, service storage.Service) error {
	m.services[service.Name] = service
	return nil
}
func (m *memoryRepo) Service(_ context.Context, name string) (storage.Service, bool, error) {
	service, ok := m.services[name]
	return service, ok, nil
}
func (m *memoryRepo) MenuServices(context.Context) ([]storage.Service, error) { return nil, nil }
func (m *memoryRepo) SetSetting(context.Context, storage.Setting) error       { return nil }
func (m *memoryRepo) GetSetting(context.Context, string) (storage.Setting, bool, error) {
	return storage.Setting{}, false, nil
}
func (m *memoryRepo) SaveClient(context.Context, storage.Client) error { return nil }
func (m *memoryRepo) ListClients(context.Context, string) ([]storage.Client, error) {
	return nil, nil
}
func (m *memoryRepo) DeleteClient(context.Context, string) error { return nil }
func (m *memoryRepo) SaveWireGuardServer(context.Context, storage.WireGuardServer) error {
	return nil
}
func (m *memoryRepo) GetWireGuardServer(context.Context, string) (storage.WireGuardServer, bool, error) {
	return storage.WireGuardServer{}, false, nil
}
func (m *memoryRepo) SetPendingOperation(context.Context, storage.PendingOperation) error {
	return nil
}
func (m *memoryRepo) GetPendingOperation(context.Context, int64) (storage.PendingOperation, bool, error) {
	return storage.PendingOperation{}, false, nil
}
func (m *memoryRepo) ClearPendingOperation(context.Context, int64) error { return nil }

type fakeCompose map[string]bool

func (f fakeCompose) EnabledServices(context.Context) (map[string]bool, error) { return f, nil }

type fakeRuntime map[string]bool

func (f fakeRuntime) RunningServices(context.Context) (map[string]bool, error) { return f, nil }

func TestRegistryHidesStoppedService(t *testing.T) {
	repo := &memoryRepo{services: map[string]storage.Service{}}
	registry := NewRegistry(repo, fakeCompose{"wg": true, "xr": true}, fakeRuntime{"wg": true})
	if err := registry.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg := repo.services["wg"]
	if !wg.Enabled || !wg.Available {
		t.Fatalf("wg should be enabled and available: %+v", wg)
	}
	xr := repo.services["xr"]
	if !xr.Enabled || xr.Available {
		t.Fatalf("xr should be enabled but unavailable: %+v", xr)
	}
}
