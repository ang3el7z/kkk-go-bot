package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrateIsVersionedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := OpenSQLite(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		table string
		name  string
	}{
		{"admins", "admins"},
		{"services", "services"},
		{"settings", "settings"},
		{"clients", "clients"},
		{"wireguard_servers", "wireguard_servers"},
		{"pending_operations", "pending_operations"},
		{"schema_migrations", "schema_migrations"},
		{"idx_services_menu", "idx_services_menu"},
		{"idx_clients_protocol_name", "idx_clients_protocol_name"},
		{"idx_pending_operations_user_expires", "idx_pending_operations_user_expires"},
		{"idx_settings_secret", "idx_settings_secret"},
	} {
		if !sqliteObjectExists(t, db, item.name) {
			t.Fatalf("missing sqlite object %s", item.table)
		}
	}
	var count int
	if err := db.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(migrations()) {
		t.Fatalf("bad migration count: %d", count)
	}
}

func sqliteObjectExists(t *testing.T, db *SQLite, name string) bool {
	t.Helper()
	var one int
	err := db.db.QueryRow(`SELECT 1 FROM sqlite_master WHERE name = ?`, name).Scan(&one)
	return err == nil
}
