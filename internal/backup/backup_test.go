package backup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestExportImportSkipsSecretsByDefault(t *testing.T) {
	ctx := context.Background()
	source, err := storage.OpenSQLite(filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := source.AddAdmin(ctx, storage.Admin{TelegramID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := source.SetSetting(ctx, storage.Setting{Key: "public", ValueJSON: `"ok"`}); err != nil {
		t.Fatal(err)
	}
	if err := source.SetSetting(ctx, storage.Setting{Key: "secret", ValueJSON: `"hide"`, Secret: true}); err != nil {
		t.Fatal(err)
	}
	data, err := Export(ctx, source, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hide") {
		t.Fatalf("secret leaked: %s", data)
	}
	target, err := storage.OpenSQLite(filepath.Join(t.TempDir(), "target.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := target.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := Import(ctx, target, data, false); err != nil {
		t.Fatal(err)
	}
	if ok, err := target.IsAdmin(ctx, 1); err != nil || !ok {
		t.Fatalf("admin missing: %v %v", ok, err)
	}
	if _, ok, err := target.GetSetting(ctx, "public"); err != nil || !ok {
		t.Fatalf("public setting missing: %v %v", ok, err)
	}
	if _, ok, err := target.GetSetting(ctx, "secret"); err != nil || ok {
		t.Fatalf("secret should not import: %v %v", ok, err)
	}
}
