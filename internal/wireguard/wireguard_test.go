package wireguard

import (
	"context"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestAddToggleDeletePeer(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	repo, err := storage.OpenSQLite(dir + "/bot.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(config.Config{
		ConfigDir: dir,
		WGAddress: "10.9.0.1/24",
		WGPort:    "51820",
		PublicIP:  "203.0.113.1",
	}, repo)
	client, conf, err := manager.Add(ctx, "wg", "test", "0.0.0.0/0")
	if err != nil {
		t.Fatal(err)
	}
	if conf == "" || client.ID == "" {
		t.Fatalf("missing client data: %+v %q", client, conf)
	}
	if err := manager.Toggle(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Rename(ctx, client.ID, "renamed"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetDNS(ctx, client.ID, "1.1.1.1,8.8.8.8"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetMTU(ctx, client.ID, "1420"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetAllowedIPs(ctx, client.ID, "10.0.0.0/8"); err != nil {
		t.Fatal(err)
	}
	clients, err := manager.List(ctx, "wg")
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 || clients[0].Enabled || clients[0].Name != "renamed" {
		t.Fatalf("client should be disabled: %+v", clients)
	}
	if err := manager.Delete(ctx, client.ID); err != nil {
		t.Fatal(err)
	}
	clients, err = manager.List(ctx, "wg")
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 0 {
		t.Fatalf("client should be deleted: %+v", clients)
	}
}
