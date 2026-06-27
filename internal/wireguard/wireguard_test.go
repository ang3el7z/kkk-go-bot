package wireguard

import (
	"context"
	"os"
	"strings"
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
	if err := os.WriteFile(dir+"/pac.json", []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(config.Config{
		ConfigDir: dir,
		WGAddress: "10.9.0.1/24",
		WGPort:    "51820",
		PublicIP:  "203.0.113.1",
	}, repo)
	if err := manager.SetDefaultAllowedIPs(ctx, "wg", "10.0.0.0/8"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ToggleEndpoint(ctx, "wg"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ToggleBlockTorrent(ctx, "wg"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ToggleBlockExchange(ctx, "wg"); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddSubnet(ctx, "wg", "172.16.0.0/12"); err != nil {
		t.Fatal(err)
	}
	pac, err := os.ReadFile(dir + "/pac.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pac), "blocktorrent") || !strings.Contains(string(pac), "exchange") {
		t.Fatalf("pac flags missing: %s", pac)
	}
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
	enabled, err := manager.ToggleAmnezia(ctx, "wg")
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("amnezia should be enabled")
	}
	info, err := manager.Info(ctx, "wg")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Amnezia || !info.EndpointUseIP || !info.BlockTorrent || !info.BlockExchange || info.DefaultAllowedIPs != "10.0.0.0/8" || len(info.Subnets) != 1 || len(info.Clients) != 1 || info.Clients[0].Address == "" {
		t.Fatalf("bad info: %+v", info)
	}
	if err := manager.DeleteSubnet(ctx, "wg", "172.16.0.0/12"); err != nil {
		t.Fatal(err)
	}
	_, png, err := manager.ClientQR(ctx, client.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("empty QR")
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

func TestParseWGShow(t *testing.T) {
	statuses := parseWGShow(`interface: wg0
peer: pubkey1
  latest handshake: 1 minute, 2 seconds ago
  transfer: 1.00 KiB received, 2.00 KiB sent
peer: pubkey2
  transfer: 3.00 KiB received, 4.00 KiB sent
`)
	if statuses["pubkey1"].Handshake == "" || statuses["pubkey1"].Transfer == "" {
		t.Fatalf("missing status: %+v", statuses)
	}
	if statuses["pubkey2"].Transfer == "" {
		t.Fatalf("missing second peer: %+v", statuses)
	}
}
