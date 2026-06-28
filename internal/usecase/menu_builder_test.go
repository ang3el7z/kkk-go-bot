package usecase

import (
	"testing"

	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
)

func TestMenuBuilderColumnsRowsAndURLs(t *testing.T) {
	builder := NewMenuBuilder(2)
	builder.Add("A", "a")
	builder.Add("B", "b")
	builder.AddURL("Docs", "https://example.com")
	builder.Row(telegram.InlineButton{Text: "Back", Data: "back"})
	keyboard := builder.Build()
	if keyboard == nil || len(keyboard.Rows) != 3 {
		t.Fatalf("bad keyboard: %+v", keyboard)
	}
	if len(keyboard.Rows[0]) != 2 || keyboard.Rows[0][0].Data != "a" || keyboard.Rows[0][1].Data != "b" {
		t.Fatalf("bad first row: %+v", keyboard.Rows[0])
	}
	if len(keyboard.Rows[1]) != 1 || keyboard.Rows[1][0].URL == "" {
		t.Fatalf("bad url row: %+v", keyboard.Rows[1])
	}
	if keyboard.Rows[2][0].Text != "Back" {
		t.Fatalf("bad manual row: %+v", keyboard.Rows[2])
	}
}

func TestMenuBuilderEmptyBuild(t *testing.T) {
	if keyboard := NewMenuBuilder(2).Build(); keyboard != nil {
		t.Fatalf("expected nil keyboard, got %+v", keyboard)
	}
}
