package backup

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type Snapshot struct {
	Version         int                       `json:"version"`
	ExportedAt      string                    `json:"exported_at"`
	Admins          []storage.Admin           `json:"admins,omitempty"`
	Services        []storage.Service         `json:"services,omitempty"`
	Settings        []storage.Setting         `json:"settings,omitempty"`
	Clients         []storage.Client          `json:"clients,omitempty"`
	WireGuardServer []storage.WireGuardServer `json:"wireguard_servers,omitempty"`
}

func Export(ctx context.Context, repo storage.Repository, includeSecrets bool) ([]byte, error) {
	admins, err := repo.ListAdmins(ctx)
	if err != nil {
		return nil, err
	}
	services, err := repo.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := repo.ListSettings(ctx, includeSecrets)
	if err != nil {
		return nil, err
	}
	clients, err := repo.ListClients(ctx, "")
	if err != nil {
		return nil, err
	}
	servers, err := repo.ListWireGuardServers(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(Snapshot{
		Version:         2,
		ExportedAt:      time.Now().UTC().Format(time.RFC3339),
		Admins:          admins,
		Services:        services,
		Settings:        settings,
		Clients:         clients,
		WireGuardServer: servers,
	}, "", "  ")
}

func Import(ctx context.Context, repo storage.Repository, data []byte, allowSecrets bool) error {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	for _, admin := range snapshot.Admins {
		if err := repo.AddAdmin(ctx, admin); err != nil {
			return err
		}
	}
	for _, service := range snapshot.Services {
		if err := repo.UpsertService(ctx, service); err != nil {
			return err
		}
	}
	for _, setting := range snapshot.Settings {
		if setting.Secret && !allowSecrets {
			continue
		}
		if err := repo.SetSetting(ctx, setting); err != nil {
			return err
		}
	}
	for _, client := range snapshot.Clients {
		if err := repo.SaveClient(ctx, client); err != nil {
			return err
		}
	}
	for _, server := range snapshot.WireGuardServer {
		if err := repo.SaveWireGuardServer(ctx, server); err != nil {
			return err
		}
	}
	return nil
}
