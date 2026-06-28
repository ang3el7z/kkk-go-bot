package xray

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestAddToggleRenameDelete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xray.json"), []byte(`{"inbounds":[{"settings":{"clients":[]}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "xray.stats"), []byte(`{"users":{"0":{"global":{"download":1024,"upload":2048},"session":{"download":1024,"upload":2048}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := storage.OpenSQLite(filepath.Join(dir, "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(config.Config{ConfigDir: dir}, repo)
	client, err := manager.Add(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Toggle(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Rename(ctx, client.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetTimer(ctx, client.ID, "2030-01-02 03:04:05"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ResetUserStats(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetTransport(ctx, "Reality"); err != nil {
		t.Fatal(err)
	}
	if enabled, err := manager.ToggleGlobalHWID(ctx); err != nil || !enabled {
		t.Fatalf("bad global hwid: %v %v", enabled, err)
	}
	if err := manager.SetDefaultHWIDLimit(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if err := manager.ToggleClientHWID(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetClientHWIDLimit(ctx, client.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddRouteItem(ctx, "block", "example.com"); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddTemplate(ctx, "v2ray", "basic", `{"outbounds":[]}`); err != nil {
		t.Fatal(err)
	}
	if err := manager.ResetUUID(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	clientsAfterReset, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(clientsAfterReset) != 1 || clientsAfterReset[0].ID == client.ID {
		t.Fatalf("uuid should reset: before=%s after=%+v", client.ID, clientsAfterReset)
	}
	client = clientsAfterReset[0]
	link, err := manager.Link(ctx, client.ID)
	if err != nil {
		t.Fatal(err)
	}
	if link == "" {
		t.Fatal("empty link")
	}
	contentType, body, err := manager.Subscription(ctx, client.ID[len("xray:"):], "si")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/json" || body == "" {
		t.Fatalf("bad subscription: %s %q", contentType, body)
	}
	_, png, err := manager.QR(ctx, client.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("empty QR")
	}
	info, err := manager.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.Transport != "Reality" || !info.HWIDEnabled || info.HWIDDefault != 3 || len(info.Routes.Block) != 1 || len(info.Templates.V2Ray) != 1 || len(info.Clients) != 1 || info.Clients[0].Download == "" || info.Clients[0].Upload == "" {
		t.Fatalf("bad info: %+v", info)
	}
	if err := manager.DeleteRouteItem(ctx, "block", "example.com"); err != nil {
		t.Fatal(err)
	}
	if err := manager.DeleteTemplate(ctx, "v2ray", "basic"); err != nil {
		t.Fatal(err)
	}
	clients, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 || clients[0].Enabled || clients[0].Name != "bob" {
		t.Fatalf("bad clients: %+v", clients)
	}
	if err := manager.Delete(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	clients, err = manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 0 {
		t.Fatalf("client should be deleted: %+v", clients)
	}
}

func TestSetStatCounters(t *testing.T) {
	values := map[string]any{
		"global": map[string]any{"download": float64(10), "upload": float64(20)},
	}
	setStatCounters(values, 5, 7)
	if nestedInt64(values, "global", "download") != 15 {
		t.Fatalf("bad download: %+v", values)
	}
	if nestedInt64(values, "session", "upload") != 7 {
		t.Fatalf("bad session: %+v", values)
	}
}

func TestSubscriptionTemplatesRedirectAndWindowsZip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xray.json"), []byte(`{"inbounds":[{"settings":{"clients":[]},"streamSettings":{"realitySettings":{"serverNames":["cdn.example"],"shortIds":["abc"]}}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pac.json"), []byte(`{"xray":"pub","outbound":"proxy","domain":"direct.example","linkdomain":"cdn.example"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sing.json"), []byte(`{"outbounds":[{"tag":"~outbound~","uuid":"~uid~","server":"~domain~"}],"route":{"rules":[{"domain_suffix":"~block~"}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := storage.OpenSQLite(filepath.Join(dir, "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(config.Config{ConfigDir: dir, Domain: "vpn.example", PublicIP: "203.0.113.10"}, repo)
	client, err := manager.Add(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Toggle(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddRouteItem(ctx, "block", "blocked.example"); err != nil {
		t.Fatal(err)
	}
	uuid := strings.TrimPrefix(client.ID, "xray:")
	contentType, body, err := manager.Subscription(ctx, uuid, "si")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/json" || !strings.Contains(body, uuid) || !strings.Contains(body, "blocked.example") || strings.Contains(body, "~uid~") {
		t.Fatalf("bad templated subscription: %s %s", contentType, body)
	}
	location, err := manager.ImportRedirect(ctx, uuid, "", "v", "https://vpn.example")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(location, "v2rayng://install-config?url=") || !strings.Contains(location, "t%3Ds") {
		t.Fatalf("bad redirect: %s", location)
	}
	name, data, err := manager.WindowsZip(ctx, uuid, "https://vpn.example")
	if err != nil {
		t.Fatal(err)
	}
	if name != "singbox.zip" || len(data) == 0 {
		t.Fatalf("bad zip: %s %d", name, len(data))
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var xml string
	for _, file := range reader.File {
		if file.Name != "winsw3.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		xml = string(body)
	}
	if !strings.Contains(xml, "https://vpn.example/pac?t=si") || !strings.Contains(xml, uuid) {
		t.Fatalf("bad zip xml: %s", xml)
	}
}
