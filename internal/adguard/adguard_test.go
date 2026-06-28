package adguard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestAddDeleteUpstream(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AdGuardHome.yaml"), []byte(`bind_host: 0.0.0.0
bind_port: 3000
protection_enabled: true
users:
  - name: admin
    password: hash
dns:
  upstream_dns:
    - 1.1.1.1
`), 0o644); err != nil {
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
	if err := manager.AddUpstream(ctx, "https://dns.google/dns-query"); err != nil {
		t.Fatal(err)
	}
	info, err := manager.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ProtectionEnabled || info.BindPort != 3000 || info.Users != 1 || len(info.Upstreams) != 2 {
		t.Fatalf("bad info: %+v", info)
	}
	setting, ok, err := repo.GetSetting(ctx, "legacy.adguard.yaml")
	if err != nil || !ok || !setting.Secret {
		t.Fatalf("adguard setting not synced: ok=%v err=%v value=%+v", ok, err, setting)
	}
	if err := manager.DeleteUpstream(ctx, "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "AdGuardHome.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "1.1.1.1") || !strings.Contains(string(body), "dns.google") {
		t.Fatalf("bad yaml after delete: %s", body)
	}
}
