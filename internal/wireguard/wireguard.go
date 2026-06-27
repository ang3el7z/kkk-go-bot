package wireguard

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type Config struct {
	Interface map[string]string   `json:"interface"`
	Peers     []map[string]string `json:"peers,omitempty"`
}

type Manager struct {
	cfg  config.Config
	repo storage.Repository
}

func NewManager(cfg config.Config, repo storage.Repository) *Manager {
	return &Manager{cfg: cfg, repo: repo}
}

func (m *Manager) List(ctx context.Context, instance string) ([]storage.Client, error) {
	return m.repo.ListClients(ctx, instance)
}

func (m *Manager) Add(ctx context.Context, instance, name, allowedIPs string) (storage.Client, string, error) {
	server, err := m.server(ctx, instance)
	if err != nil {
		return storage.Client{}, "", err
	}
	if name == "" {
		name = fmt.Sprintf("all%d", time.Now().Unix())
	}
	if allowedIPs == "" {
		allowedIPs = "0.0.0.0/0"
	}
	clientIP, err := nextClientIP(server)
	if err != nil {
		return storage.Client{}, "", err
	}
	serverPrivate := server.Interface["PrivateKey"]
	if serverPrivate == "" {
		serverPrivate, err = privateKey()
		if err != nil {
			return storage.Client{}, "", err
		}
		server.Interface["PrivateKey"] = serverPrivate
	}
	serverPublic, err := publicKey(serverPrivate)
	if err != nil {
		return storage.Client{}, "", err
	}
	clientPrivate, err := privateKey()
	if err != nil {
		return storage.Client{}, "", err
	}
	clientPublic, err := publicKey(clientPrivate)
	if err != nil {
		return storage.Client{}, "", err
	}

	server.Peers = append(server.Peers, map[string]string{
		"## name":    name,
		"PublicKey":  clientPublic,
		"AllowedIPs": clientIP.String() + "/32",
	})
	endpoint := m.endpoint(instance)
	clientConfig := Config{
		Interface: map[string]string{
			"## name":    name,
			"PrivateKey": clientPrivate,
			"Address":    clientIP.String() + "/32",
		},
		Peers: []map[string]string{{
			"PublicKey":           serverPublic,
			"AllowedIPs":          allowedIPs,
			"Endpoint":            endpoint,
			"PersistentKeepalive": "20",
		}},
	}
	body, err := json.Marshal(clientConfig)
	if err != nil {
		return storage.Client{}, "", err
	}
	client := storage.Client{
		ID:         fmt.Sprintf("%s:%s", instance, clientIP.String()),
		Protocol:   instance,
		Name:       name,
		Enabled:    true,
		ConfigJSON: string(body),
	}
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return storage.Client{}, "", err
	}
	if err := m.saveServer(ctx, instance, server); err != nil {
		return storage.Client{}, "", err
	}
	return client, Render(clientConfig), nil
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	server, err := m.server(ctx, instance)
	if err != nil {
		return err
	}
	address := cfg.Interface["Address"]
	server.Peers = filterPeers(server.Peers, address)
	if err := m.repo.DeleteClient(ctx, id); err != nil {
		return err
	}
	return m.saveServer(ctx, instance, server)
}

func (m *Manager) Toggle(ctx context.Context, id string) error {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	client.Enabled = !client.Enabled
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.rebuildServerPeers(ctx, instance)
}

func (m *Manager) Rename(ctx context.Context, id, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("WireGuard client name is empty")
	}
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	cfg.Interface["## name"] = name
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	client.Name = name
	client.ConfigJSON = string(body)
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.rebuildServerPeers(ctx, instance)
}

func (m *Manager) SetExpiry(ctx context.Context, id, expiresAt string) error {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	if strings.TrimSpace(expiresAt) == "" || strings.TrimSpace(expiresAt) == "0" {
		delete(cfg.Interface, "## time")
	} else {
		if _, err := time.Parse("2006-01-02 15:04:05", expiresAt); err != nil {
			return fmt.Errorf("time format must be YYYY-MM-DD HH:MM:SS")
		}
		cfg.Interface["## time"] = expiresAt
	}
	return m.saveClientConfigAndRebuild(ctx, instance, client, cfg)
}

func (m *Manager) SetDNS(ctx context.Context, id, dns string) error {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	if strings.TrimSpace(dns) == "" || strings.TrimSpace(dns) == "0" {
		delete(cfg.Interface, "DNS")
	} else {
		cfg.Interface["DNS"] = strings.TrimSpace(dns)
	}
	return m.saveClientConfigAndRebuild(ctx, instance, client, cfg)
}

func (m *Manager) SetMTU(ctx context.Context, id, mtu string) error {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	if strings.TrimSpace(mtu) == "" || strings.TrimSpace(mtu) == "0" {
		delete(cfg.Interface, "MTU")
	} else {
		cfg.Interface["MTU"] = strings.TrimSpace(mtu)
	}
	return m.saveClientConfigAndRebuild(ctx, instance, client, cfg)
}

func (m *Manager) SetAllowedIPs(ctx context.Context, id, allowedIPs string) error {
	if strings.TrimSpace(allowedIPs) == "" {
		return errors.New("AllowedIPs is empty")
	}
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return err
	}
	if len(cfg.Peers) == 0 {
		return errors.New("WireGuard client has no peer")
	}
	cfg.Peers[0]["AllowedIPs"] = strings.TrimSpace(allowedIPs)
	return m.saveClientConfigAndRebuild(ctx, instance, client, cfg)
}

func (m *Manager) ClientConfig(ctx context.Context, id string) (string, string, error) {
	instance, _, ok := strings.Cut(id, ":")
	if !ok {
		return "", "", errors.New("invalid WireGuard client id")
	}
	client, err := m.clientByID(ctx, instance, id)
	if err != nil {
		return "", "", err
	}
	cfg, err := decodeConfig(client.ConfigJSON)
	if err != nil {
		return "", "", err
	}
	return safeName(client.Name) + ".conf", Render(cfg), nil
}

func (m *Manager) saveClientConfigAndRebuild(ctx context.Context, instance string, client storage.Client, cfg Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	client.ConfigJSON = string(body)
	if err := m.repo.SaveClient(ctx, client); err != nil {
		return err
	}
	return m.rebuildServerPeers(ctx, instance)
}

func (m *Manager) server(ctx context.Context, instance string) (Config, error) {
	saved, ok, err := m.repo.GetWireGuardServer(ctx, instance)
	if err != nil {
		return Config{}, err
	}
	if ok {
		cfg, err := decodeConfig(saved.ConfigJSON)
		if err == nil && cfg.Interface != nil {
			return cfg, nil
		}
	}
	address := m.cfg.WGAddress
	port := m.cfg.WGPort
	if instance == "wg1" {
		address = m.cfg.WG1Address
		port = m.cfg.WG1Port
	}
	key, err := privateKey()
	if err != nil {
		return Config{}, err
	}
	return Config{Interface: map[string]string{
		"PrivateKey": key,
		"Address":    address,
		"ListenPort": port,
	}}, nil
}

func (m *Manager) saveServer(ctx context.Context, instance string, cfg Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := m.repo.SaveWireGuardServer(ctx, storage.WireGuardServer{Instance: instance, ConfigJSON: string(body)}); err != nil {
		return err
	}
	path := filepath.Join(m.cfg.ConfigDir, "wg0.conf")
	if instance == "wg1" {
		path = filepath.Join(m.cfg.ConfigDir, "wg1.conf")
	}
	if err := os.WriteFile(path, []byte(Render(cfg)), 0o600); err != nil {
		return err
	}
	if m.cfg.ReloadWG {
		return reload(ctx, instance)
	}
	return nil
}

func (m *Manager) rebuildServerPeers(ctx context.Context, instance string) error {
	server, err := m.server(ctx, instance)
	if err != nil {
		return err
	}
	server.Peers = nil
	clients, err := m.repo.ListClients(ctx, instance)
	if err != nil {
		return err
	}
	for _, client := range clients {
		cfg, err := decodeConfig(client.ConfigJSON)
		if err != nil || len(cfg.Peers) == 0 {
			continue
		}
		public, err := publicKey(cfg.Interface["PrivateKey"])
		if err != nil {
			continue
		}
		peer := map[string]string{
			"## name":    client.Name,
			"PublicKey":  public,
			"AllowedIPs": cfg.Interface["Address"],
		}
		if expiry := cfg.Interface["## time"]; expiry != "" {
			peer["## time"] = expiry
		}
		if !client.Enabled {
			peer = commentPeer(peer)
		}
		server.Peers = append(server.Peers, peer)
	}
	return m.saveServer(ctx, instance, server)
}

func (m *Manager) clientByID(ctx context.Context, instance, id string) (storage.Client, error) {
	clients, err := m.repo.ListClients(ctx, instance)
	if err != nil {
		return storage.Client{}, err
	}
	for _, client := range clients {
		if client.ID == id {
			return client, nil
		}
	}
	return storage.Client{}, errors.New("WireGuard client not found")
}

func (m *Manager) endpoint(instance string) string {
	host := m.cfg.Domain
	if host == "" {
		host = m.cfg.PublicIP
	}
	port := m.cfg.WGPort
	if instance == "wg1" {
		port = m.cfg.WG1Port
	}
	return host + ":" + port
}

func decodeConfig(raw string) (Config, error) {
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Interface == nil {
		cfg.Interface = map[string]string{}
	}
	return cfg, nil
}

func Render(cfg Config) string {
	var lines []string
	lines = append(lines, "[Interface]")
	for _, key := range sortedKeys(cfg.Interface) {
		lines = append(lines, fmt.Sprintf("%s = %s", key, cfg.Interface[key]))
	}
	for _, peer := range cfg.Peers {
		lines = append(lines, "", "[Peer]")
		for _, key := range sortedKeys(peer) {
			lines = append(lines, fmt.Sprintf("%s = %s", key, peer[key]))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func nextClientIP(server Config) (netip.Addr, error) {
	prefix, err := netip.ParsePrefix(server.Interface["Address"])
	if err != nil {
		return netip.Addr{}, err
	}
	used := map[netip.Addr]bool{prefix.Addr(): true}
	for _, peer := range server.Peers {
		raw := peer["AllowedIPs"]
		if raw == "" {
			raw = peer["# AllowedIPs"]
		}
		addr, err := firstAddr(raw)
		if err == nil {
			used[addr] = true
		}
	}
	base := prefix.Addr().As4()
	ones := prefix.Bits()
	size := uint32(1) << (32 - uint(ones))
	start := uint32(base[0])<<24 | uint32(base[1])<<16 | uint32(base[2])<<8 | uint32(base[3])
	for offset := uint32(1); offset < size-1; offset++ {
		ip := start + offset
		addr := netip.AddrFrom4([4]byte{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)})
		if prefix.Contains(addr) && !used[addr] {
			return addr, nil
		}
	}
	return netip.Addr{}, errors.New("WireGuard subnet is full")
}

func firstAddr(raw string) (netip.Addr, error) {
	part := strings.TrimSpace(strings.Split(raw, ",")[0])
	if strings.Contains(part, "/") {
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			return netip.Addr{}, err
		}
		return prefix.Addr(), nil
	}
	return netip.ParseAddr(part)
}

func filterPeers(peers []map[string]string, address string) []map[string]string {
	out := peers[:0]
	for _, peer := range peers {
		if peer["AllowedIPs"] != address && peer["# AllowedIPs"] != address {
			out = append(out, peer)
		}
	}
	return out
}

func commentPeer(peer map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range peer {
		if strings.HasPrefix(key, "#") {
			out[key] = value
		} else {
			out["# "+key] = value
		}
	}
	return out
}

func privateKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	raw[0] &= 248
	raw[31] &= 127
	raw[31] |= 64
	return base64.StdEncoding.EncodeToString(raw), nil
}

func publicKey(private string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(private)
	if err != nil {
		return "", err
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

func reload(ctx context.Context, instance string) error {
	service := instance
	cmd := fmt.Sprintf("docker compose exec -T %s sh -lc 'wg syncconf wg0 <(wg-quick strip wg0)'", service)
	return exec.CommandContext(ctx, "sh", "-lc", cmd).Run()
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "wireguard"
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

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return keyOrder(keys[i]) < keyOrder(keys[j])
	})
	return keys
}

func keyOrder(key string) string {
	order := map[string]string{
		"## name":             "00",
		"PrivateKey":          "01",
		"Address":             "02",
		"ListenPort":          "03",
		"DNS":                 "04",
		"MTU":                 "05",
		"PublicKey":           "06",
		"AllowedIPs":          "07",
		"Endpoint":            "08",
		"PersistentKeepalive": "09",
	}
	if v, ok := order[key]; ok {
		return v + key
	}
	return "99" + key
}
