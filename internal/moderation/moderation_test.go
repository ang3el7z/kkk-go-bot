package moderation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestDenyAndLogs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logs := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "nginx.log"), []byte("line1\nline2\n"), 0o644); err != nil {
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
	manager := NewManager(config.Config{ConfigDir: dir, LogsDir: logs}, repo)
	if err := manager.AddDeny(ctx, "203.0.113.10"); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddDeny(ctx, "203.0.113.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddDeny(ctx, "bad"); err == nil {
		t.Fatal("expected invalid IP error")
	}
	info, err := manager.Info()
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Deny) != 2 || len(info.Logs) != 1 {
		t.Fatalf("bad info: %+v", info)
	}
	tail, err := manager.TailLog("nginx.log", 6)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tail, "line2") {
		t.Fatalf("bad tail: %q", tail)
	}
	if err := manager.DeleteDeny(ctx, "203.0.113.10"); err != nil {
		t.Fatal(err)
	}
	deny, err := manager.DenyList()
	if err != nil {
		t.Fatal(err)
	}
	if len(deny) != 1 || deny[0] != "203.0.113.0/24" {
		t.Fatalf("bad deny list: %+v", deny)
	}
	if err := manager.ClearLogs(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(logs, "nginx.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("log not cleared: %q", body)
	}
}
