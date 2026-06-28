package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type Manager struct {
	cfg  config.Config
	repo storage.Repository
}

type Info struct {
	Deny []string
	Logs []LogFile
}

type LogFile struct {
	Name string
	Size int64
}

func NewManager(cfg config.Config, repo storage.Repository) *Manager {
	return &Manager{cfg: cfg, repo: repo}
}

func (m *Manager) Info() (Info, error) {
	deny, err := m.DenyList()
	if err != nil {
		return Info{}, err
	}
	logs, err := m.LogFiles()
	if err != nil {
		return Info{}, err
	}
	return Info{Deny: deny, Logs: logs}, nil
}

func (m *Manager) DenyList() ([]string, error) {
	data, err := os.ReadFile(m.denyPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	values := map[string]bool{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		line = strings.TrimPrefix(line, "deny ")
		line = strings.TrimSuffix(line, ";")
		line = strings.TrimSpace(line)
		if line != "" {
			values[line] = true
		}
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out, nil
}

func (m *Manager) AddDeny(ctx context.Context, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("IP/CIDR is empty")
	}
	if _, err := netip.ParseAddr(value); err != nil {
		if _, prefixErr := netip.ParsePrefix(value); prefixErr != nil {
			return errors.New("invalid IP/CIDR")
		}
	}
	values, err := m.DenyList()
	if err != nil {
		return err
	}
	for _, existing := range values {
		if existing == value {
			return nil
		}
	}
	values = append(values, value)
	return m.writeDeny(ctx, values)
}

func (m *Manager) DeleteDeny(ctx context.Context, value string) error {
	values, err := m.DenyList()
	if err != nil {
		return err
	}
	var out []string
	for _, existing := range values {
		if existing != value {
			out = append(out, existing)
		}
	}
	return m.writeDeny(ctx, out)
}

func (m *Manager) ClearDeny(ctx context.Context) error {
	return m.writeDeny(ctx, nil)
}

func (m *Manager) writeDeny(ctx context.Context, values []string) error {
	sort.Strings(values)
	var lines []string
	for _, value := range values {
		lines = append(lines, "deny "+value+";")
	}
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	if err := os.MkdirAll(filepath.Dir(m.denyPath()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.denyPath(), []byte(body), 0o644); err != nil {
		return err
	}
	value, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return m.repo.SetSetting(ctx, storage.Setting{Key: "legacy.deny", ValueJSON: string(value)})
}

func (m *Manager) LogFiles() ([]LogFile, error) {
	entries, err := os.ReadDir(m.cfg.LogsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []LogFile
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, LogFile{Name: entry.Name(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *Manager) TailLog(name string, maxBytes int64) (string, error) {
	if strings.Contains(name, "/") || strings.Contains(name, `\`) || name == "" {
		return "", errors.New("bad log name")
	}
	data, err := os.ReadFile(filepath.Join(m.cfg.LogsDir, name))
	if err != nil {
		return "", err
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		data = data[int64(len(data))-maxBytes:]
	}
	return string(data), nil
}

func (m *Manager) ClearLogs() error {
	entries, err := os.ReadDir(m.cfg.LogsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if err := os.WriteFile(filepath.Join(m.cfg.LogsDir, entry.Name()), nil, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) denyPath() string {
	return filepath.Join(m.cfg.ConfigDir, "deny")
}
