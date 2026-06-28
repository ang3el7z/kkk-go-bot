package legacy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type Importer struct {
	cfg  config.Config
	repo storage.Repository
}

func NewImporter(cfg config.Config, repo storage.Repository) *Importer {
	return &Importer{cfg: cfg, repo: repo}
}

func (i *Importer) Import(ctx context.Context) error {
	if err := i.importPHPConfig(ctx); err != nil {
		return err
	}
	if err := i.importJSONSetting(ctx, "pac", filepath.Join(i.cfg.ConfigDir, "pac.json")); err != nil {
		return err
	}
	if err := i.importClients(ctx, "wg", filepath.Join(i.cfg.ConfigDir, "clients.json")); err != nil {
		return err
	}
	if err := i.importClients(ctx, "wg1", filepath.Join(i.cfg.ConfigDir, "clients1.json")); err != nil {
		return err
	}
	if err := i.importWGConfig(ctx, "wg", filepath.Join(i.cfg.ConfigDir, "wg0.conf")); err != nil {
		return err
	}
	if err := i.importWGConfig(ctx, "wg1", filepath.Join(i.cfg.ConfigDir, "wg1.conf")); err != nil {
		return err
	}
	if err := i.importJSONSetting(ctx, "hwid", filepath.Join(i.cfg.ConfigDir, "hwid.json")); err != nil {
		return err
	}
	if err := i.importXray(ctx, filepath.Join(i.cfg.ConfigDir, "xray.json")); err != nil {
		return err
	}
	return nil
}

func (i *Importer) importWGConfig(ctx context.Context, instance, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	parsed := parseWGConfig(string(data))
	if len(parsed) == 0 {
		return nil
	}
	body, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	return i.repo.SaveWireGuardServer(ctx, storage.WireGuardServer{Instance: instance, ConfigJSON: string(body)})
}

func (i *Importer) importPHPConfig(ctx context.Context) error {
	data, err := os.ReadFile(i.cfg.LegacyPHPPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	text := string(data)
	for _, id := range parseAdminIDs(text) {
		if err := i.repo.AddAdmin(ctx, storage.Admin{TelegramID: id}); err != nil {
			return err
		}
	}
	for key, value := range parsePHPScalars(text) {
		if isSecretKey(key) {
			continue
		}
		if err := i.repo.SetSetting(ctx, storage.Setting{Key: "legacy.config_php." + key, ValueJSON: mustJSON(value)}); err != nil {
			return err
		}
	}
	redacted := redactPHPConfig(text)
	return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy.config_php.redacted", ValueJSON: mustJSON(redacted)})
}

func (i *Importer) importJSONSetting(ctx context.Context, key, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !json.Valid(data) {
		return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy." + key + ".raw", ValueJSON: mustJSON(string(data))})
	}
	return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy." + key, ValueJSON: string(data)})
}

func (i *Importer) importXray(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !json.Valid(data) {
		return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy.xray.raw", ValueJSON: mustJSON(string(data))})
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	for _, client := range xrayClients(doc) {
		id, _ := client["id"].(string)
		email, _ := client["email"].(string)
		if id == "" || email == "" {
			continue
		}
		body, _ := json.Marshal(client)
		if err := i.repo.SaveClient(ctx, storage.Client{
			ID:         "xray:" + id,
			Protocol:   "xray",
			Name:       email,
			Enabled:    client["off"] == nil,
			ConfigJSON: string(body),
		}); err != nil {
			return err
		}
	}
	return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy.xray", ValueJSON: string(data)})
}

func (i *Importer) importClients(ctx context.Context, protocol, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !json.Valid(data) {
		return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy." + protocol + ".clients.raw", ValueJSON: mustJSON(string(data))})
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for name, value := range legacyClientItems(payload) {
		body, err := json.Marshal(value)
		if err != nil {
			return err
		}
		client := storage.Client{
			ID:         fmt.Sprintf("%s:%s", protocol, name),
			Protocol:   protocol,
			Name:       name,
			Enabled:    true,
			ConfigJSON: string(body),
		}
		if err := i.repo.SaveClient(ctx, client); err != nil {
			return err
		}
	}
	return i.repo.SetSetting(ctx, storage.Setting{Key: "legacy." + protocol + ".clients", ValueJSON: string(data)})
}

func legacyClientItems(payload any) map[string]any {
	items := map[string]any{}
	switch v := payload.(type) {
	case map[string]any:
		for name, value := range v {
			items[name] = value
		}
	case []any:
		for idx, value := range v {
			name := fmt.Sprintf("%d", idx)
			if obj, ok := value.(map[string]any); ok {
				if rawName, ok := obj["name"].(string); ok && rawName != "" {
					name = rawName
				}
			}
			items[name] = value
		}
	}
	return items
}

func xrayClients(doc map[string]any) []map[string]any {
	inbounds, _ := doc["inbounds"].([]any)
	if len(inbounds) == 0 {
		return nil
	}
	inbound, _ := inbounds[0].(map[string]any)
	settings, _ := inbound["settings"].(map[string]any)
	rawClients, _ := settings["clients"].([]any)
	clients := make([]map[string]any, 0, len(rawClients))
	for _, raw := range rawClients {
		client, ok := raw.(map[string]any)
		if ok {
			clients = append(clients, client)
		}
	}
	return clients
}

func parseAdminIDs(text string) []int64 {
	re := regexp.MustCompile(`['"]admin['"]\s*=>\s*\[([^\]]*)\]`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return nil
	}
	numRe := regexp.MustCompile(`\d+`)
	var ids []int64
	for _, raw := range numRe.FindAllString(match[1], -1) {
		var id int64
		_, _ = fmt.Sscan(raw, &id)
		if id != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func redactPHPConfig(text string) string {
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		lower := strings.ToLower(line)
		for _, key := range []string{"key", "token", "password", "passwd", "secret"} {
			if strings.Contains(lower, key) && strings.Contains(line, "=>") {
				lines[idx] = regexp.MustCompile(`=>\s*['"][^'"]*['"]`).ReplaceAllString(line, `=> '***REDACTED***'`)
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func parsePHPScalars(text string) map[string]string {
	re := regexp.MustCompile(`['"]([A-Za-z0-9_.-]+)['"]\s*=>\s*(?:['"]([^'"]*)['"]|(-?\d+)|\b(true|false)\b)`)
	values := map[string]string{}
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		value := match[2]
		if value == "" {
			value = match[3]
		}
		if value == "" {
			value = match[4]
		}
		if match[1] != "" && value != "" {
			values[match[1]] = value
		}
	}
	return values
}

func isSecretKey(key string) bool {
	key = strings.ToLower(key)
	for _, part := range []string{"key", "token", "password", "passwd", "secret"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func mustJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func parseWGConfig(text string) map[string]any {
	result := map[string]any{}
	var peers []map[string]string
	current := map[string]string{}
	section := ""
	flush := func() {
		if section == "Peer" && len(current) > 0 {
			peers = append(peers, current)
		}
		current = map[string]string{}
	}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			section = strings.Trim(line, "[]")
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if section == "Interface" {
			if _, ok := result["interface"]; !ok {
				result["interface"] = map[string]string{}
			}
			result["interface"].(map[string]string)[key] = value
		} else if section == "Peer" {
			current[key] = value
		}
	}
	flush()
	if len(peers) > 0 {
		result["peers"] = peers
	}
	return result
}
