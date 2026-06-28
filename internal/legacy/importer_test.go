package legacy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

func TestPHPConfigParsingRedactsSecretsAndImportsSafeScalars(t *testing.T) {
	text := `<?php
$c = [
    'key' => 'telegram-token',
    'admin' => [123, 456],
    'debug' => true,
    'lang' => 'en',
    'backup_days' => 7,
    'api_secret' => 'hide-me',
];`
	admins := parseAdminIDs(text)
	if len(admins) != 2 || admins[0] != 123 || admins[1] != 456 {
		t.Fatalf("bad admins: %+v", admins)
	}
	scalars := parsePHPScalars(text)
	if scalars["lang"] != "en" || scalars["debug"] != "true" || scalars["backup_days"] != "7" {
		t.Fatalf("bad scalars: %+v", scalars)
	}
	if !isSecretKey("api_secret") || !isSecretKey("key") || isSecretKey("lang") {
		t.Fatal("bad secret key detection")
	}
	redacted := redactPHPConfig(text)
	if strings.Contains(redacted, "telegram-token") || strings.Contains(redacted, "hide-me") {
		t.Fatalf("secret leaked: %s", redacted)
	}
	if !strings.Contains(redacted, "***REDACTED***") {
		t.Fatalf("missing redaction: %s", redacted)
	}
}

func TestImporterReadsLegacyPHPConfigSafely(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	phpPath := filepath.Join(dir, "config.php")
	writeFile(t, phpPath, `<?php $c = ['key' => 'telegram-token', 'admin' => [777], 'lang' => 'ru'];`)
	writeFile(t, filepath.Join(dir, "pac.json"), `{}`)
	writeFile(t, filepath.Join(dir, "hwid.json"), `{}`)
	writeFile(t, filepath.Join(dir, "xray.json"), `{"inbounds":[{"settings":{"clients":[]}}]}`)
	writeFile(t, filepath.Join(dir, "xray.stats"), `{"users":{}}`)
	writeFile(t, filepath.Join(dir, "mtprotosecret"), `secret-value`)
	writeFile(t, filepath.Join(dir, "mtprotodomain"), `tg.example`)
	writeFile(t, filepath.Join(dir, "ssserver.json"), `{"password":"secret"}`)
	certs := filepath.Join(dir, "certs")
	if err := os.MkdirAll(certs, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(certs, "cert_private"), `private-key`)

	repo, err := storage.OpenSQLite(filepath.Join(dir, "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	err = NewImporter(config.Config{ConfigDir: dir, CertsDir: certs, LegacyPHPPath: phpPath}, repo).Import(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.IsAdmin(ctx, 777); err != nil || !ok {
		t.Fatalf("admin not imported: %v %v", ok, err)
	}
	lang, ok, err := repo.GetSetting(ctx, "legacy.config_php.lang")
	if err != nil || !ok || lang.ValueJSON != `"ru"` {
		t.Fatalf("lang not imported: ok=%v err=%v value=%+v", ok, err, lang)
	}
	if _, ok, err := repo.GetSetting(ctx, "legacy.config_php.key"); err != nil || ok {
		t.Fatalf("secret key imported: ok=%v err=%v", ok, err)
	}
	mtproto, ok, err := repo.GetSetting(ctx, "legacy.mtprotosecret")
	if err != nil || !ok || !mtproto.Secret || mtproto.ValueJSON != `"secret-value"` {
		t.Fatalf("mtproto secret not imported safely: ok=%v err=%v value=%+v", ok, err, mtproto)
	}
	domain, ok, err := repo.GetSetting(ctx, "legacy.mtprotodomain")
	if err != nil || !ok || domain.Secret || domain.ValueJSON != `"tg.example"` {
		t.Fatalf("mtproto domain not imported: ok=%v err=%v value=%+v", ok, err, domain)
	}
	ss, ok, err := repo.GetSetting(ctx, "legacy.ssserver.json")
	if err != nil || !ok || !ss.Secret || !strings.Contains(ss.ValueJSON, "password") {
		t.Fatalf("ss config not imported as secret: ok=%v err=%v value=%+v", ok, err, ss)
	}
	cert, ok, err := repo.GetSetting(ctx, "legacy.cert_private")
	if err != nil || !ok || !cert.Secret {
		t.Fatalf("cert private not imported as secret: ok=%v err=%v value=%+v", ok, err, cert)
	}
	redacted, ok, err := repo.GetSetting(ctx, "legacy.config_php.redacted")
	if err != nil || !ok || strings.Contains(redacted.ValueJSON, "telegram-token") {
		t.Fatalf("bad redacted config: ok=%v err=%v value=%+v", ok, err, redacted)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
