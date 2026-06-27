package xray

import (
	"context"
	"os"
	"path/filepath"
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
	if len(info) != 1 || info[0].Download == "" || info[0].Upload == "" {
		t.Fatalf("bad info: %+v", info)
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
