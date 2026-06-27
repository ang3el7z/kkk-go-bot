package xray

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	qrcode "github.com/skip2/go-qrcode"
)

type Manager struct {
	cfg  config.Config
	repo storage.Repository
}

func NewManager(cfg config.Config, repo storage.Repository) *Manager {
	return &Manager{cfg: cfg, repo: repo}
}

func (m *Manager) List(ctx context.Context) ([]storage.Client, error) {
	return m.repo.ListClients(ctx, "xray")
}

func (m *Manager) Add(ctx context.Context, email string) (storage.Client, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return storage.Client{}, errors.New("Xray user email/name is empty")
	}
	id, err := uuid()
	if err != nil {
		return storage.Client{}, err
	}
	body, _ := json.Marshal(map[string]any{"id": id, "email": email})
	client := storage.Client{ID: "xray:" + id, Protocol: "xray", Name: email, Enabled: true, ConfigJSON: string(body)}
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return storage.Client{}, err
	}
	return client, m.Render(ctx)
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	if err := m.repo.DeleteClient(ctx, id); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) Toggle(ctx context.Context, id string) error {
	client, err := m.client(ctx, id)
	if err != nil {
		return err
	}
	client.Enabled = !client.Enabled
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) Rename(ctx context.Context, id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("Xray name is empty")
	}
	client, err := m.client(ctx, id)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return err
	}
	payload["email"] = name
	body, _ := json.Marshal(payload)
	client.Name = name
	client.ConfigJSON = string(body)
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) SetTimer(ctx context.Context, id, expiresAt string) error {
	client, err := m.client(ctx, id)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return err
	}
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" || expiresAt == "0" {
		delete(payload, "time")
	} else {
		t, err := time.Parse("2006-01-02 15:04:05", expiresAt)
		if err != nil {
			return fmt.Errorf("time format must be YYYY-MM-DD HH:MM:SS")
		}
		payload["time"] = t.Unix()
	}
	body, _ := json.Marshal(payload)
	client.ConfigJSON = string(body)
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) ResetUserStats(ctx context.Context, id string) error {
	return m.repo.SetSetting(ctx, storage.Setting{Key: "xray.stats.reset." + id, ValueJSON: fmt.Sprintf("%q", time.Now().UTC().Format(time.RFC3339))})
}

func (m *Manager) Link(ctx context.Context, id string) (string, error) {
	client, err := m.client(ctx, id)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return "", err
	}
	uuid, _ := payload["id"].(string)
	host := m.cfg.Domain
	if host == "" {
		host = m.cfg.PublicIP
	}
	if host == "" {
		host = "example.com"
	}
	path := "/ws"
	return fmt.Sprintf("vless://%s@%s:443?encryption=none&security=tls&type=ws&path=%s#%s", uuid, host, path, client.Name), nil
}

func (m *Manager) QR(ctx context.Context, id string) (string, []byte, error) {
	link, err := m.Link(ctx, id)
	if err != nil {
		return "", nil, err
	}
	client, err := m.client(ctx, id)
	if err != nil {
		return "", nil, err
	}
	png, err := qrcode.Encode(link, qrcode.Medium, 512)
	if err != nil {
		return "", nil, err
	}
	return safeName(client.Name) + ".png", png, nil
}

func (m *Manager) Render(ctx context.Context) error {
	doc, err := m.template()
	if err != nil {
		return err
	}
	clients, err := m.repo.ListClients(ctx, "xray")
	if err != nil {
		return err
	}
	rendered := make([]any, 0, len(clients))
	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
			return err
		}
		delete(payload, "off")
		rendered = append(rendered, payload)
	}
	if err := setClients(doc, rendered); err != nil {
		return err
	}
	body, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.cfg.ConfigDir, "xray.json"), body, 0o644)
}

func (m *Manager) client(ctx context.Context, id string) (storage.Client, error) {
	clients, err := m.repo.ListClients(ctx, "xray")
	if err != nil {
		return storage.Client{}, err
	}
	for _, client := range clients {
		if client.ID == id {
			return client, nil
		}
	}
	return storage.Client{}, errors.New("Xray user not found")
}

func (m *Manager) template() (map[string]any, error) {
	path := filepath.Join(m.cfg.ConfigDir, "xray.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func setClients(doc map[string]any, clients []any) error {
	inbounds, _ := doc["inbounds"].([]any)
	if len(inbounds) == 0 {
		return errors.New("xray config has no inbounds")
	}
	inbound, _ := inbounds[0].(map[string]any)
	settings, _ := inbound["settings"].(map[string]any)
	if settings == nil {
		return errors.New("xray inbound has no settings")
	}
	settings["clients"] = clients
	return nil
}

func uuid() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "xray"
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
