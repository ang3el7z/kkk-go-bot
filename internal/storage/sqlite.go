package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

type migration struct {
	version    int
	name       string
	statements []string
}

func OpenSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	for _, migration := range migrations() {
		applied, err := s.migrationApplied(ctx, migration.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := s.applyMigration(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) migrationApplied(ctx context.Context, version int) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM schema_migrations WHERE version = ?`, version).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *SQLite) applyMigration(ctx context.Context, migration migration) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, statement := range migration.statements {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)`, migration.version, migration.name, now()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrations() []migration {
	return []migration{
		{
			version: 1,
			name:    "initial_go_runtime",
			statements: []string{
				`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER NOT NULL UNIQUE,
			username TEXT NOT NULL DEFAULT '',
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
				`CREATE TABLE IF NOT EXISTS services (
			name TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			enabled INTEGER NOT NULL,
			available INTEGER NOT NULL,
			availability_reason TEXT NOT NULL DEFAULT '',
			menu_group TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
				`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			secret INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
				`CREATE TABLE IF NOT EXISTS clients (
			id TEXT PRIMARY KEY,
			protocol TEXT NOT NULL,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(protocol, name)
		)`,
				`CREATE TABLE IF NOT EXISTS wireguard_servers (
			instance TEXT PRIMARY KEY,
			config_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL
		)`,
				`CREATE TABLE IF NOT EXISTS pending_operations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER NOT NULL,
			operation TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
			},
		},
		{
			version: 2,
			name:    "query_indexes",
			statements: []string{
				`CREATE INDEX IF NOT EXISTS idx_services_menu ON services(enabled, available, menu_group, sort_order, name)`,
				`CREATE INDEX IF NOT EXISTS idx_clients_protocol_name ON clients(protocol, name)`,
				`CREATE INDEX IF NOT EXISTS idx_pending_operations_user_expires ON pending_operations(telegram_id, expires_at)`,
				`CREATE INDEX IF NOT EXISTS idx_settings_secret ON settings(secret, key)`,
			},
		},
	}
}

func (s *SQLite) AddAdmin(ctx context.Context, admin Admin) error {
	created := admin.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO admins (telegram_id, username, first_name, last_name, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(telegram_id) DO UPDATE SET username = excluded.username, first_name = excluded.first_name, last_name = excluded.last_name`,
		admin.TelegramID, admin.Username, admin.FirstName, admin.LastName, timeString(created))
	return err
}

func (s *SQLite) HasAdmins(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count)
	return count > 0, err
}

func (s *SQLite) IsAdmin(ctx context.Context, telegramID int64) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM admins WHERE telegram_id = ?`, telegramID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *SQLite) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, telegram_id, username, first_name, last_name, created_at FROM admins ORDER BY telegram_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var admins []Admin
	for rows.Next() {
		var admin Admin
		var created string
		if err := rows.Scan(&admin.ID, &admin.TelegramID, &admin.Username, &admin.FirstName, &admin.LastName, &created); err != nil {
			return nil, err
		}
		admin.CreatedAt, _ = parseTime(created)
		admins = append(admins, admin)
	}
	return admins, rows.Err()
}

func (s *SQLite) UpsertService(ctx context.Context, service Service) error {
	if service.UpdatedAt.IsZero() {
		service.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO services (name, display_name, enabled, available, availability_reason, menu_group, sort_order, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
	display_name = excluded.display_name,
	enabled = excluded.enabled,
	available = excluded.available,
	availability_reason = excluded.availability_reason,
	menu_group = excluded.menu_group,
	sort_order = excluded.sort_order,
	updated_at = excluded.updated_at`,
		service.Name, service.DisplayName, boolInt(service.Enabled), boolInt(service.Available), service.AvailabilityReason,
		service.MenuGroup, service.SortOrder, timeString(service.UpdatedAt))
	return err
}

func (s *SQLite) Service(ctx context.Context, name string) (Service, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT name, display_name, enabled, available, availability_reason, menu_group, sort_order, updated_at FROM services WHERE name = ?`, name)
	service, err := scanService(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Service{}, false, nil
	}
	return service, err == nil, err
}

func (s *SQLite) ListServices(ctx context.Context) ([]Service, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, display_name, enabled, available, availability_reason, menu_group, sort_order, updated_at FROM services ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var services []Service
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

func (s *SQLite) MenuServices(ctx context.Context) ([]Service, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT name, display_name, enabled, available, availability_reason, menu_group, sort_order, updated_at
FROM services
WHERE enabled = 1 AND available = 1 AND menu_group = 'main'
ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var services []Service
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

func (s *SQLite) SetSetting(ctx context.Context, setting Setting) error {
	if setting.UpdatedAt.IsZero() {
		setting.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO settings (key, value_json, secret, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, secret = excluded.secret, updated_at = excluded.updated_at`,
		setting.Key, setting.ValueJSON, boolInt(setting.Secret), timeString(setting.UpdatedAt))
	return err
}

func (s *SQLite) GetSetting(ctx context.Context, key string) (Setting, bool, error) {
	var setting Setting
	var secret int
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT key, value_json, secret, updated_at FROM settings WHERE key = ?`, key).Scan(&setting.Key, &setting.ValueJSON, &secret, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Setting{}, false, nil
	}
	if err != nil {
		return Setting{}, false, err
	}
	setting.Secret = secret != 0
	setting.UpdatedAt, _ = parseTime(updated)
	return setting, true, nil
}

func (s *SQLite) ListSettings(ctx context.Context, includeSecrets bool) ([]Setting, error) {
	query := `SELECT key, value_json, secret, updated_at FROM settings`
	if !includeSecrets {
		query += ` WHERE secret = 0`
	}
	query += ` ORDER BY key`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var settings []Setting
	for rows.Next() {
		var setting Setting
		var secret int
		var updated string
		if err := rows.Scan(&setting.Key, &setting.ValueJSON, &secret, &updated); err != nil {
			return nil, err
		}
		setting.Secret = secret != 0
		setting.UpdatedAt, _ = parseTime(updated)
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (s *SQLite) SaveClient(ctx context.Context, client Client) error {
	nowTime := time.Now().UTC()
	if client.CreatedAt.IsZero() {
		client.CreatedAt = nowTime
	}
	client.UpdatedAt = nowTime
	if client.ConfigJSON == "" {
		client.ConfigJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO clients (id, protocol, name, enabled, config_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET protocol = excluded.protocol, name = excluded.name, enabled = excluded.enabled, config_json = excluded.config_json, updated_at = excluded.updated_at`,
		client.ID, client.Protocol, client.Name, boolInt(client.Enabled), client.ConfigJSON, timeString(client.CreatedAt), timeString(client.UpdatedAt))
	return err
}

func (s *SQLite) ListClients(ctx context.Context, protocol string) ([]Client, error) {
	query := `SELECT id, protocol, name, enabled, config_json, created_at, updated_at FROM clients`
	var args []any
	if protocol != "" {
		query += ` WHERE protocol = ?`
		args = append(args, protocol)
	}
	query += ` ORDER BY protocol, name`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var clients []Client
	for rows.Next() {
		var client Client
		var enabled int
		var created, updated string
		if err := rows.Scan(&client.ID, &client.Protocol, &client.Name, &enabled, &client.ConfigJSON, &created, &updated); err != nil {
			return nil, err
		}
		client.Enabled = enabled != 0
		client.CreatedAt, _ = parseTime(created)
		client.UpdatedAt, _ = parseTime(updated)
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (s *SQLite) DeleteClient(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM clients WHERE id = ?`, id)
	return err
}

func (s *SQLite) SaveWireGuardServer(ctx context.Context, server WireGuardServer) error {
	if server.UpdatedAt.IsZero() {
		server.UpdatedAt = time.Now().UTC()
	}
	if server.ConfigJSON == "" {
		server.ConfigJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO wireguard_servers (instance, config_json, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(instance) DO UPDATE SET config_json = excluded.config_json, updated_at = excluded.updated_at`,
		server.Instance, server.ConfigJSON, timeString(server.UpdatedAt))
	return err
}

func (s *SQLite) GetWireGuardServer(ctx context.Context, instance string) (WireGuardServer, bool, error) {
	var server WireGuardServer
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT instance, config_json, updated_at FROM wireguard_servers WHERE instance = ?`, instance).Scan(&server.Instance, &server.ConfigJSON, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return WireGuardServer{}, false, nil
	}
	if err != nil {
		return WireGuardServer{}, false, err
	}
	server.UpdatedAt, _ = parseTime(updated)
	return server, true, nil
}

func (s *SQLite) ListWireGuardServers(ctx context.Context) ([]WireGuardServer, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT instance, config_json, updated_at FROM wireguard_servers ORDER BY instance`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []WireGuardServer
	for rows.Next() {
		var server WireGuardServer
		var updated string
		if err := rows.Scan(&server.Instance, &server.ConfigJSON, &updated); err != nil {
			return nil, err
		}
		server.UpdatedAt, _ = parseTime(updated)
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

func (s *SQLite) SetPendingOperation(ctx context.Context, op PendingOperation) error {
	nowTime := time.Now().UTC()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = nowTime
	}
	if op.ExpiresAt.IsZero() {
		op.ExpiresAt = nowTime.Add(15 * time.Minute)
	}
	if op.PayloadJSON == "" {
		op.PayloadJSON = "{}"
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM pending_operations WHERE telegram_id = ?`, op.TelegramID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO pending_operations (telegram_id, operation, payload_json, created_at, expires_at)
VALUES (?, ?, ?, ?, ?)`,
		op.TelegramID, op.Operation, op.PayloadJSON, timeString(op.CreatedAt), timeString(op.ExpiresAt))
	return err
}

func (s *SQLite) GetPendingOperation(ctx context.Context, telegramID int64) (PendingOperation, bool, error) {
	var op PendingOperation
	var created, expires string
	err := s.db.QueryRowContext(ctx, `
SELECT id, telegram_id, operation, payload_json, created_at, expires_at
FROM pending_operations
WHERE telegram_id = ? AND expires_at > ?`,
		telegramID, now()).Scan(&op.ID, &op.TelegramID, &op.Operation, &op.PayloadJSON, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return PendingOperation{}, false, nil
	}
	if err != nil {
		return PendingOperation{}, false, err
	}
	op.CreatedAt, _ = parseTime(created)
	op.ExpiresAt, _ = parseTime(expires)
	return op, true, nil
}

func (s *SQLite) ClearPendingOperation(ctx context.Context, telegramID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pending_operations WHERE telegram_id = ?`, telegramID)
	return err
}

type scanner interface{ Scan(dest ...any) error }

func scanService(row scanner) (Service, error) {
	var service Service
	var enabled, available int
	var updated string
	if err := row.Scan(&service.Name, &service.DisplayName, &enabled, &available, &service.AvailabilityReason, &service.MenuGroup, &service.SortOrder, &updated); err != nil {
		return Service{}, err
	}
	service.Enabled = enabled != 0
	service.Available = available != 0
	service.UpdatedAt, _ = parseTime(updated)
	return service, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func now() string { return timeString(time.Now().UTC()) }

func timeString(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", v, err)
	}
	return t.UTC(), nil
}
