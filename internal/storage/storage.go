package storage

import (
	"context"
	"time"
)

type Admin struct {
	ID         int64
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	CreatedAt  time.Time
}

type Service struct {
	Name               string
	DisplayName        string
	Enabled            bool
	Available          bool
	AvailabilityReason string
	MenuGroup          string
	SortOrder          int
	UpdatedAt          time.Time
}

type Setting struct {
	Key       string
	ValueJSON string
	Secret    bool
	UpdatedAt time.Time
}

type Client struct {
	ID         string
	Protocol   string
	Name       string
	Enabled    bool
	ConfigJSON string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type WireGuardServer struct {
	Instance   string
	ConfigJSON string
	UpdatedAt  time.Time
}

type PendingOperation struct {
	ID          int64
	TelegramID  int64
	Operation   string
	PayloadJSON string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type Repository interface {
	Migrate(ctx context.Context) error
	Close() error
	AddAdmin(ctx context.Context, admin Admin) error
	HasAdmins(ctx context.Context) (bool, error)
	IsAdmin(ctx context.Context, telegramID int64) (bool, error)
	ListAdmins(ctx context.Context) ([]Admin, error)
	UpsertService(ctx context.Context, service Service) error
	Service(ctx context.Context, name string) (Service, bool, error)
	MenuServices(ctx context.Context) ([]Service, error)
	SetSetting(ctx context.Context, setting Setting) error
	GetSetting(ctx context.Context, key string) (Setting, bool, error)
	SaveClient(ctx context.Context, client Client) error
	ListClients(ctx context.Context, protocol string) ([]Client, error)
	DeleteClient(ctx context.Context, id string) error
	SaveWireGuardServer(ctx context.Context, server WireGuardServer) error
	GetWireGuardServer(ctx context.Context, instance string) (WireGuardServer, bool, error)
	SetPendingOperation(ctx context.Context, op PendingOperation) error
	GetPendingOperation(ctx context.Context, telegramID int64) (PendingOperation, bool, error)
	ClearPendingOperation(ctx context.Context, telegramID int64) error
}
