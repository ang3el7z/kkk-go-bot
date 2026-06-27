package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
)

type Bot struct {
	repo storage.Repository
}

func NewBot(repo storage.Repository) *Bot {
	return &Bot{repo: repo}
}

type MessageResult struct {
	Text     string
	Keyboard *telegram.InlineKeyboard
}

type CallbackResult struct {
	Text      string
	ShowAlert bool
}

func (b *Bot) HandleMessage(ctx context.Context, msg telegram.Message) (MessageResult, error) {
	if msg.Text == "/id" {
		return MessageResult{Text: fmt.Sprintf("User ID: %d\nChat ID: %d", msg.From.ID, msg.Chat.ID)}, nil
	}
	if err := b.requireAdminOrBootstrap(ctx, msg.From); err != nil {
		return MessageResult{Text: "Unauthorized"}, nil
	}
	switch msg.Text {
	case "/start", "/menu", "":
		return b.menu(ctx)
	default:
		return MessageResult{Text: "Unknown command. Use /menu."}, nil
	}
}

func (b *Bot) HandleCallback(ctx context.Context, query telegram.CallbackQuery) (CallbackResult, error) {
	if err := b.requireAdminOrBootstrap(ctx, query.From); err != nil {
		return CallbackResult{Text: "Unauthorized", ShowAlert: true}, nil
	}
	name, ok := strings.CutPrefix(query.Data, "service:")
	if !ok {
		return CallbackResult{Text: "Route not migrated yet", ShowAlert: true}, nil
	}
	service, found, err := b.repo.Service(ctx, name)
	if err != nil {
		return CallbackResult{}, err
	}
	if !found || !service.Enabled || !service.Available {
		reason := service.AvailabilityReason
		if reason == "" {
			reason = "service unavailable"
		}
		return CallbackResult{Text: reason, ShowAlert: true}, nil
	}
	return CallbackResult{Text: service.DisplayName, ShowAlert: false}, nil
}

func (b *Bot) requireAdminOrBootstrap(ctx context.Context, user telegram.User) error {
	hasAdmins, err := b.repo.HasAdmins(ctx)
	if err != nil {
		return err
	}
	if !hasAdmins {
		return b.repo.AddAdmin(ctx, storage.Admin{
			TelegramID: user.ID,
			Username:   user.Username,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
		})
	}
	ok, err := b.repo.IsAdmin(ctx, user.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unauthorized")
	}
	return nil
}

func (b *Bot) menu(ctx context.Context) (MessageResult, error) {
	services, err := b.repo.MenuServices(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	keyboard := &telegram.InlineKeyboard{}
	for i := 0; i < len(services); i += 2 {
		row := []telegram.InlineButton{{
			Text: services[i].DisplayName,
			Data: "service:" + services[i].Name,
		}}
		if i+1 < len(services) {
			row = append(row, telegram.InlineButton{
				Text: services[i+1].DisplayName,
				Data: "service:" + services[i+1].Name,
			})
		}
		keyboard.Rows = append(keyboard.Rows, row)
	}
	if len(keyboard.Rows) == 0 {
		return MessageResult{Text: "No enabled services found in compose."}, nil
	}
	return MessageResult{Text: "kkk-go-bot menu", Keyboard: keyboard}, nil
}
