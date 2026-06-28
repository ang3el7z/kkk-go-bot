package xray

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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

type ClientInfo struct {
	ID           string
	Name         string
	Enabled      bool
	Download     string
	Upload       string
	HWIDDisabled bool
	HWIDLimit    int
}

type Info struct {
	Transport   string
	HWIDEnabled bool
	HWIDDefault int
	Clients     []ClientInfo
	Routes      RouteLists
	Templates   TemplateInfo
}

type RouteLists struct {
	Block    []string
	Warp     []string
	Proxy    []string
	Subnet   []string
	Process  []string
	Package  []string
	RuleSets []string
}

type TemplateInfo struct {
	V2Ray []string
	Sing  []string
	Clash []string
}

func NewManager(cfg config.Config, repo storage.Repository) *Manager {
	return &Manager{cfg: cfg, repo: repo}
}

func (m *Manager) List(ctx context.Context) ([]storage.Client, error) {
	return m.repo.ListClients(ctx, "xray")
}

func (m *Manager) Info(ctx context.Context) (Info, error) {
	clients, err := m.repo.ListClients(ctx, "xray")
	if err != nil {
		return Info{}, err
	}
	stats := m.readStats()
	transport, err := m.Transport(ctx)
	if err != nil {
		return Info{}, err
	}
	hwidEnabled, err := m.boolSetting(ctx, "xray.hwid.enabled")
	if err != nil {
		return Info{}, err
	}
	hwidDefault, err := m.intSetting(ctx, "xray.hwid.default", 1)
	if err != nil {
		return Info{}, err
	}
	info := Info{Transport: transport, HWIDEnabled: hwidEnabled, HWIDDefault: hwidDefault, Clients: make([]ClientInfo, 0, len(clients))}
	info.Routes = m.routes(ctx)
	info.Templates = m.templates(ctx)
	for idx, client := range clients {
		item := ClientInfo{ID: client.ID, Name: client.Name, Enabled: client.Enabled}
		var payload map[string]any
		_ = json.Unmarshal([]byte(client.ConfigJSON), &payload)
		item.HWIDDisabled, _ = payload["hwid_disabled"].(bool)
		if limit, ok := payload["hwid_limit"].(float64); ok {
			item.HWIDLimit = int(limit)
		}
		if userStats := stats.user(idx); userStats != nil {
			item.Download = bytesLabel(userStats.download())
			item.Upload = bytesLabel(userStats.upload())
		}
		info.Clients = append(info.Clients, item)
	}
	return info, nil
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

func (m *Manager) ResetUUID(ctx context.Context, id string) error {
	client, err := m.client(ctx, id)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return err
	}
	newID, err := uuid()
	if err != nil {
		return err
	}
	payload["id"] = newID
	body, _ := json.Marshal(payload)
	client.ID = "xray:" + newID
	client.ConfigJSON = string(body)
	if err := m.repo.DeleteClient(ctx, id); err != nil {
		return err
	}
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) Transport(ctx context.Context) (string, error) {
	setting, ok, err := m.repo.GetSetting(ctx, "xray.transport")
	if err != nil {
		return "", err
	}
	if ok {
		var transport string
		if err := json.Unmarshal([]byte(setting.ValueJSON), &transport); err == nil && transport != "" {
			return transport, nil
		}
	}
	return "Websocket", nil
}

func (m *Manager) SetTransport(ctx context.Context, transport string) error {
	switch transport {
	case "Websocket", "Reality", "xhttp":
	default:
		return errors.New("unsupported Xray transport")
	}
	body, _ := json.Marshal(transport)
	if err := m.repo.SetSetting(ctx, storage.Setting{Key: "xray.transport", ValueJSON: string(body)}); err != nil {
		return err
	}
	if err := m.setPacValue("transport", transport); err != nil {
		return err
	}
	return m.Render(ctx)
}

func (m *Manager) ToggleGlobalHWID(ctx context.Context) (bool, error) {
	enabled, err := m.toggleBoolSetting(ctx, "xray.hwid.enabled")
	if err != nil {
		return false, err
	}
	if err := m.setPacValue("hwid_limit_enabled", enabled); err != nil {
		return false, err
	}
	return enabled, nil
}

func (m *Manager) SetDefaultHWIDLimit(ctx context.Context, count int) error {
	if count < 1 {
		count = 1
	}
	if err := m.repo.SetSetting(ctx, storage.Setting{Key: "xray.hwid.default", ValueJSON: fmt.Sprintf("%d", count)}); err != nil {
		return err
	}
	return m.setPacValue("hwid_device_count", count)
}

func (m *Manager) ToggleClientHWID(ctx context.Context, id string) error {
	return m.updateClientPayload(ctx, id, func(payload map[string]any) {
		disabled, _ := payload["hwid_disabled"].(bool)
		if disabled {
			delete(payload, "hwid_disabled")
		} else {
			payload["hwid_disabled"] = true
		}
	})
}

func (m *Manager) SetClientHWIDLimit(ctx context.Context, id string, count int) error {
	return m.updateClientPayload(ctx, id, func(payload map[string]any) {
		if count < 1 {
			delete(payload, "hwid_limit")
		} else {
			payload["hwid_limit"] = count
		}
	})
}

func (m *Manager) AddRouteItem(ctx context.Context, list, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("route value is empty")
	}
	values, err := m.stringList(ctx, routeSettingKey(list))
	if err != nil {
		return err
	}
	for _, existing := range values {
		if existing == value {
			return nil
		}
	}
	values = append(values, value)
	return m.setStringList(ctx, routeSettingKey(list), values)
}

func (m *Manager) DeleteRouteItem(ctx context.Context, list, value string) error {
	values, err := m.stringList(ctx, routeSettingKey(list))
	if err != nil {
		return err
	}
	out := values[:0]
	for _, existing := range values {
		if existing != value {
			out = append(out, existing)
		}
	}
	return m.setStringList(ctx, routeSettingKey(list), out)
}

func (m *Manager) AddTemplate(ctx context.Context, typ, name, body string) error {
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	if name == "" || body == "" {
		return errors.New("template name/body is empty")
	}
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return err
	}
	templates, err := m.templateMap(ctx, typ)
	if err != nil {
		return err
	}
	templates[name] = parsed
	return m.setTemplateMap(ctx, typ, templates)
}

func (m *Manager) DeleteTemplate(ctx context.Context, typ, name string) error {
	templates, err := m.templateMap(ctx, typ)
	if err != nil {
		return err
	}
	delete(templates, name)
	return m.setTemplateMap(ctx, typ, templates)
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
	return fmt.Sprintf("vless://%s@%s:443?encryption=none&security=tls&type=ws&path=/ws#%s", uuid, m.host(), client.Name), nil
}

func (m *Manager) Subscription(ctx context.Context, uuid, typ string) (string, string, error) {
	client, err := m.clientByUUID(ctx, uuid)
	if err != nil {
		return "", "", err
	}
	link, err := m.Link(ctx, client.ID)
	if err != nil {
		return "", "", err
	}
	switch typ {
	case "si":
		body, err := m.subscriptionTemplate(ctx, client, "sing")
		if err != nil {
			return "", "", err
		}
		if body == "" {
			body = fmt.Sprintf(`{"outbounds":[{"type":"vless","tag":"%s","server":"%s","server_port":443,"uuid":"%s","tls":{"enabled":true},"transport":{"type":"ws","path":"/ws"}}]}`+"\n", client.Name, m.host(), uuid)
		}
		return "application/json", body, nil
	case "cl":
		body, err := m.subscriptionTemplate(ctx, client, "clash")
		if err != nil {
			return "", "", err
		}
		if body == "" {
			body = fmt.Sprintf("proxies:\n  - name: %s\n    type: vless\n    server: %s\n    port: 443\n    uuid: %s\n    tls: true\n    network: ws\n    ws-opts:\n      path: /ws\n", client.Name, m.host(), uuid)
		}
		return "text/yaml", body, nil
	default:
		body, err := m.subscriptionTemplate(ctx, client, "v2ray")
		if err != nil {
			return "", "", err
		}
		if body == "" {
			body = link + "\n"
		}
		return "text/plain", body, nil
	}
}

func (m *Manager) subscriptionTemplate(ctx context.Context, client storage.Client, typ string) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return "", err
	}
	uuid, _ := payload["id"].(string)
	doc, err := m.subscriptionTemplateDoc(ctx, typ)
	if err != nil {
		return "", err
	}
	if doc == nil {
		return "", nil
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	text := string(body)
	routes := m.routes(ctx)
	listTags := map[string][]string{
		`"~pac~"`:     routes.Proxy,
		`"~block~"`:   routes.Block,
		`"~warp~"`:    routes.Warp,
		`"~process~"`: routes.Process,
		`"~package~"`: routes.Package,
		`"~subnet~"`:  routes.Subnet,
	}
	for tag, values := range listTags {
		encoded, _ := json.Marshal(values)
		text = strings.ReplaceAll(text, tag, string(encoded))
	}
	reality := m.realitySettings()
	pac := m.pacMap()
	stringTags := map[string]string{
		"~outbound~":     pacString(pac, "outbound", "proxy"),
		"~dns~":          "https://" + m.host() + "/dns-query/" + uuid,
		"~dnspath~":      "/dns-query/" + uuid,
		"~uid~":          uuid,
		"~domain~":       m.host(),
		"~directdomain~": pacString(pac, "domain", m.host()),
		"~cdndomain~":    pacString(pac, "linkdomain", m.host()),
		"~short_id~":     reality.shortID,
		"~email~":        client.Name,
		"~public_key~":   pacString(pac, "xray", ""),
		"~server_name~":  reality.serverName,
		"~ip~":           m.cfg.PublicIP,
	}
	for tag, value := range stringTags {
		text = strings.ReplaceAll(text, tag, value)
	}
	var out any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		return "", err
	}
	return string(pretty) + "\n", nil
}

func (m *Manager) subscriptionTemplateDoc(ctx context.Context, typ string) (any, error) {
	templates, err := m.templateMap(ctx, typ)
	if err != nil {
		return nil, err
	}
	if len(templates) > 0 {
		keys := sortedMapKeys(templates)
		return templates[keys[0]], nil
	}
	name := typ
	if typ == "sing" {
		name = "sing"
	}
	data, err := os.ReadFile(filepath.Join(m.cfg.ConfigDir, name+".json"))
	if err != nil {
		return nil, nil
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (m *Manager) ImportRedirect(ctx context.Context, uuid, typ, redirect, baseURL string) (string, error) {
	client, err := m.clientByUUID(ctx, uuid)
	if err != nil {
		return "", err
	}
	if typ == "" {
		typ = "s"
	}
	v2 := baseURL + "/pac?t=s&s=" + url.QueryEscape(uuid)
	si := baseURL + "/pac?t=si&s=" + url.QueryEscape(uuid)
	cl := baseURL + "/pac?t=cl&s=" + url.QueryEscape(uuid)
	switch redirect {
	case "si":
		return "sing-box://import-remote-profile/?url=" + url.QueryEscape(si), nil
	case "st":
		return "streisand://import/" + url.QueryEscape(v2), nil
	case "v":
		return "v2rayng://install-config?url=" + url.QueryEscape(v2), nil
	case "k":
		return "karing://install-config?url=" + url.QueryEscape(si), nil
	case "h":
		return "hiddify://install-config/?url=" + url.QueryEscape(si), nil
	case "c":
		return "clash://install-config/?url=" + url.QueryEscape(cl) + "&overwrite=no&name=" + url.QueryEscape(client.Name), nil
	default:
		return "", fmt.Errorf("unknown import redirect %q", redirect)
	}
}

func (m *Manager) WindowsZip(ctx context.Context, uuid, baseURL string) (string, []byte, error) {
	if _, err := m.clientByUUID(ctx, uuid); err != nil {
		return "", nil, err
	}
	source, err := readSingboxAsset("singbox.zip")
	if err != nil {
		return "", nil, err
	}
	xml, err := readSingboxAsset("winsw3.xml")
	if err != nil {
		return "", nil, err
	}
	link := html.EscapeString(baseURL + "/pac?t=si&s=" + url.QueryEscape(uuid))
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	reader, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if err != nil {
		return "", nil, err
	}
	for _, file := range reader.File {
		if file.Name == "winsw3.xml" {
			continue
		}
		if err := copyZipFile(writer, file); err != nil {
			_ = writer.Close()
			return "", nil, err
		}
	}
	w, err := writer.Create("winsw3.xml")
	if err != nil {
		_ = writer.Close()
		return "", nil, err
	}
	_, _ = w.Write([]byte(strings.ReplaceAll(string(xml), "~url~", link)))
	if err := writer.Close(); err != nil {
		return "", nil, err
	}
	return "singbox.zip", buf.Bytes(), nil
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
	transport, err := m.Transport(ctx)
	if err != nil {
		return err
	}
	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
			return err
		}
		delete(payload, "off")
		if transport == "Reality" {
			payload["flow"] = "xtls-rprx-vision"
		} else {
			delete(payload, "flow")
		}
		rendered = append(rendered, payload)
	}
	if err := setClients(doc, rendered); err != nil {
		return err
	}
	setTransport(doc, transport)
	body, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.cfg.ConfigDir, "xray.json"), body, 0o644)
}

func (m *Manager) updateClientPayload(ctx context.Context, id string, update func(map[string]any)) error {
	client, err := m.client(ctx, id)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
		return err
	}
	update(payload)
	body, _ := json.Marshal(payload)
	client.ConfigJSON = string(body)
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.Render(ctx)
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

func (m *Manager) clientByUUID(ctx context.Context, uuid string) (storage.Client, error) {
	clients, err := m.repo.ListClients(ctx, "xray")
	if err != nil {
		return storage.Client{}, err
	}
	for _, client := range clients {
		var payload map[string]any
		if err := json.Unmarshal([]byte(client.ConfigJSON), &payload); err != nil {
			continue
		}
		if payload["id"] == uuid {
			return client, nil
		}
	}
	return storage.Client{}, errors.New("Xray user not found")
}

func (m *Manager) host() string {
	host := m.cfg.Domain
	if host == "" {
		host = m.cfg.PublicIP
	}
	if host == "" {
		host = "example.com"
	}
	return host
}

func (m *Manager) boolSetting(ctx context.Context, key string) (bool, error) {
	setting, ok, err := m.repo.GetSetting(ctx, key)
	if err != nil || !ok {
		return false, err
	}
	return setting.ValueJSON == "true", nil
}

func (m *Manager) toggleBoolSetting(ctx context.Context, key string) (bool, error) {
	enabled, err := m.boolSetting(ctx, key)
	if err != nil {
		return false, err
	}
	enabled = !enabled
	return enabled, m.repo.SetSetting(ctx, storage.Setting{Key: key, ValueJSON: fmt.Sprintf("%t", enabled)})
}

func (m *Manager) intSetting(ctx context.Context, key string, fallback int) (int, error) {
	setting, ok, err := m.repo.GetSetting(ctx, key)
	if err != nil || !ok {
		return fallback, err
	}
	var value int
	if _, err := fmt.Sscanf(setting.ValueJSON, "%d", &value); err != nil {
		return fallback, nil
	}
	if value < 1 {
		return fallback, nil
	}
	return value, nil
}

func (m *Manager) setPacValue(key string, value any) error {
	path := filepath.Join(m.cfg.ConfigDir, "pac.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pac := map[string]any{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &pac)
	}
	pac[key] = value
	body, err := json.MarshalIndent(pac, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func (m *Manager) pacMap() map[string]any {
	data, err := os.ReadFile(filepath.Join(m.cfg.ConfigDir, "pac.json"))
	if err != nil {
		return map[string]any{}
	}
	pac := map[string]any{}
	_ = json.Unmarshal(data, &pac)
	return pac
}

type realityInfo struct {
	shortID    string
	serverName string
}

func (m *Manager) realitySettings() realityInfo {
	doc, err := m.template()
	if err != nil {
		return realityInfo{serverName: "www.cloudflare.com"}
	}
	inbounds, _ := doc["inbounds"].([]any)
	if len(inbounds) == 0 {
		return realityInfo{serverName: "www.cloudflare.com"}
	}
	inbound, _ := inbounds[0].(map[string]any)
	stream, _ := inbound["streamSettings"].(map[string]any)
	reality, _ := stream["realitySettings"].(map[string]any)
	info := realityInfo{serverName: "www.cloudflare.com"}
	if names, _ := reality["serverNames"].([]any); len(names) > 0 {
		if value, _ := names[0].(string); value != "" {
			info.serverName = value
		}
	}
	if ids, _ := reality["shortIds"].([]any); len(ids) > 0 {
		info.shortID, _ = ids[0].(string)
	}
	return info
}

func pacString(pac map[string]any, key, fallback string) string {
	value, _ := pac[key].(string)
	if value == "" {
		return fallback
	}
	return value
}

func (m *Manager) routes(ctx context.Context) RouteLists {
	return RouteLists{
		Block:    mustList(m.stringList(ctx, routeSettingKey("block"))),
		Warp:     mustList(m.stringList(ctx, routeSettingKey("warp"))),
		Proxy:    mustList(m.stringList(ctx, routeSettingKey("proxy"))),
		Subnet:   mustList(m.stringList(ctx, routeSettingKey("subnet"))),
		Process:  mustList(m.stringList(ctx, routeSettingKey("process"))),
		Package:  mustList(m.stringList(ctx, routeSettingKey("package"))),
		RuleSets: mustList(m.stringList(ctx, routeSettingKey("ruleset"))),
	}
}

func (m *Manager) templates(ctx context.Context) TemplateInfo {
	return TemplateInfo{
		V2Ray: sortedMapKeys(mustMap(m.templateMap(ctx, "v2ray"))),
		Sing:  sortedMapKeys(mustMap(m.templateMap(ctx, "sing"))),
		Clash: sortedMapKeys(mustMap(m.templateMap(ctx, "clash"))),
	}
}

func (m *Manager) stringList(ctx context.Context, key string) ([]string, error) {
	setting, ok, err := m.repo.GetSetting(ctx, key)
	if err != nil || !ok {
		return nil, err
	}
	var values []string
	if err := json.Unmarshal([]byte(setting.ValueJSON), &values); err != nil {
		return nil, nil
	}
	return values, nil
}

func (m *Manager) setStringList(ctx context.Context, key string, values []string) error {
	sort.Strings(values)
	body, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return m.repo.SetSetting(ctx, storage.Setting{Key: key, ValueJSON: string(body)})
}

func (m *Manager) templateMap(ctx context.Context, typ string) (map[string]any, error) {
	setting, ok, err := m.repo.GetSetting(ctx, templateSettingKey(typ))
	if err != nil || !ok {
		return map[string]any{}, err
	}
	values := map[string]any{}
	if err := json.Unmarshal([]byte(setting.ValueJSON), &values); err != nil {
		return map[string]any{}, nil
	}
	return values, nil
}

func (m *Manager) setTemplateMap(ctx context.Context, typ string, values map[string]any) error {
	body, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return m.repo.SetSetting(ctx, storage.Setting{Key: templateSettingKey(typ), ValueJSON: string(body)})
}

func mustList(values []string, _ error) []string {
	return values
}

func mustMap(values map[string]any, _ error) map[string]any {
	return values
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func routeSettingKey(list string) string {
	return "xray.routes." + list
}

func templateSettingKey(typ string) string {
	return "xray.templates." + typ
}

func copyZipFile(writer *zip.Writer, file *zip.File) error {
	header := file.FileHeader
	w, err := writer.CreateHeader(&header)
	if err != nil {
		return err
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

func readSingboxAsset(name string) ([]byte, error) {
	candidates := []string{filepath.Join("/singbox", name), filepath.Join("singbox_windows", name)}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			candidates = append(candidates, filepath.Join(dir, "singbox_windows", name))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	var last error
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		last = err
	}
	return nil, last
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

func setTransport(doc map[string]any, transport string) {
	inbounds, _ := doc["inbounds"].([]any)
	if len(inbounds) == 0 {
		return
	}
	inbound, _ := inbounds[0].(map[string]any)
	switch transport {
	case "Reality":
		inbound["streamSettings"] = map[string]any{
			"network":  "tcp",
			"security": "reality",
			"realitySettings": map[string]any{
				"show":        false,
				"dest":        "www.cloudflare.com:443",
				"serverNames": []any{"www.cloudflare.com"},
				"shortIds":    []any{""},
			},
		}
	case "xhttp":
		inbound["streamSettings"] = map[string]any{
			"network":  "xhttp",
			"security": "tls",
			"xhttpSettings": map[string]any{
				"path": "/ws",
			},
		}
	default:
		inbound["streamSettings"] = map[string]any{
			"network": "ws",
			"wsSettings": map[string]any{
				"path": "/ws",
			},
		}
	}
}

type xrayStats map[string]any

func (m *Manager) readStats() xrayStats {
	data, err := os.ReadFile(filepath.Join(m.cfg.ConfigDir, "xray.stats"))
	if err != nil {
		return nil
	}
	var stats xrayStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil
	}
	return stats
}

func (s xrayStats) user(idx int) xrayStats {
	users, _ := s["users"].(map[string]any)
	if users == nil {
		return nil
	}
	user, _ := users[fmt.Sprintf("%d", idx)].(map[string]any)
	if user == nil {
		return nil
	}
	return xrayStats(user)
}

func (s xrayStats) download() int64 {
	return nestedInt64(s, "global", "download") + nestedInt64(s, "session", "download")
}

func (s xrayStats) upload() int64 {
	return nestedInt64(s, "global", "upload") + nestedInt64(s, "session", "upload")
}

func nestedInt64(values map[string]any, group, key string) int64 {
	nested, _ := values[group].(map[string]any)
	if nested == nil {
		return 0
	}
	switch value := nested[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	default:
		return 0
	}
}

func bytesLabel(v int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(v)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
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
