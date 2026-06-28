package adguard

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	cfg  config.Config
	repo storage.Repository
}

type Info struct {
	ProtectionEnabled bool
	BindHost          string
	BindPort          int
	Users             int
	Upstreams         []string
}

func NewManager(cfg config.Config, repo storage.Repository) *Manager {
	return &Manager{cfg: cfg, repo: repo}
}

func (m *Manager) Info(ctx context.Context) (Info, error) {
	doc, err := m.read()
	if err != nil {
		return Info{}, err
	}
	info := Info{}
	info.ProtectionEnabled, _ = doc["protection_enabled"].(bool)
	info.BindHost, _ = doc["bind_host"].(string)
	if port, ok := doc["bind_port"].(int); ok {
		info.BindPort = port
	}
	if users, ok := doc["users"].([]any); ok {
		info.Users = len(users)
	}
	info.Upstreams = upstreams(doc)
	return info, nil
}

func (m *Manager) AddUpstream(ctx context.Context, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("upstream is empty")
	}
	doc, err := m.read()
	if err != nil {
		return err
	}
	values := upstreams(doc)
	for _, existing := range values {
		if existing == value {
			return nil
		}
	}
	values = append(values, value)
	return m.setUpstreams(ctx, doc, values)
}

func (m *Manager) DeleteUpstream(ctx context.Context, value string) error {
	doc, err := m.read()
	if err != nil {
		return err
	}
	var values []string
	for _, existing := range upstreams(doc) {
		if existing != value {
			values = append(values, existing)
		}
	}
	return m.setUpstreams(ctx, doc, values)
}

func (m *Manager) setUpstreams(ctx context.Context, doc map[string]any, values []string) error {
	sort.Strings(values)
	dns, _ := doc["dns"].(map[string]any)
	if dns == nil {
		dns = map[string]any{}
		doc["dns"] = dns
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	dns["upstream_dns"] = out
	body, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.path(), body, 0o644); err != nil {
		return err
	}
	value, err := json.Marshal(string(body))
	if err != nil {
		return err
	}
	return m.repo.SetSetting(ctx, storage.Setting{Key: "legacy.adguard.yaml", ValueJSON: string(value), Secret: true})
}

func (m *Manager) read() (map[string]any, error) {
	data, err := os.ReadFile(m.path())
	if err != nil {
		return nil, err
	}
	doc := map[string]any{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (m *Manager) path() string {
	return filepath.Join(m.cfg.ConfigDir, "AdGuardHome.yaml")
}

func upstreams(doc map[string]any) []string {
	dns, _ := doc["dns"].(map[string]any)
	if dns == nil {
		return nil
	}
	raw, _ := dns["upstream_dns"].([]any)
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			values = append(values, strings.TrimSpace(value))
		}
	}
	sort.Strings(values)
	return values
}
