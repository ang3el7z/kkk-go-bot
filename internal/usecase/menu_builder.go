package usecase

import "github.com/ang3el7z/kkk-go-bot/internal/telegram"

type MenuBuilder struct {
	columns int
	rows    [][]telegram.InlineButton
	current []telegram.InlineButton
}

func NewMenuBuilder(columns int) *MenuBuilder {
	if columns <= 0 {
		columns = 1
	}
	return &MenuBuilder{columns: columns}
}

func (b *MenuBuilder) Add(text, data string) {
	b.AddButton(telegram.InlineButton{Text: text, Data: data})
}

func (b *MenuBuilder) AddURL(text, url string) {
	b.AddButton(telegram.InlineButton{Text: text, URL: url})
}

func (b *MenuBuilder) AddButton(button telegram.InlineButton) {
	if button.Text == "" {
		return
	}
	b.current = append(b.current, button)
	if len(b.current) >= b.columns {
		b.Flush()
	}
}

func (b *MenuBuilder) Row(buttons ...telegram.InlineButton) {
	b.Flush()
	row := make([]telegram.InlineButton, 0, len(buttons))
	for _, button := range buttons {
		if button.Text != "" {
			row = append(row, button)
		}
	}
	if len(row) > 0 {
		b.rows = append(b.rows, row)
	}
}

func (b *MenuBuilder) Flush() {
	if len(b.current) == 0 {
		return
	}
	row := make([]telegram.InlineButton, len(b.current))
	copy(row, b.current)
	b.rows = append(b.rows, row)
	b.current = nil
}

func (b *MenuBuilder) Build() *telegram.InlineKeyboard {
	b.Flush()
	if len(b.rows) == 0 {
		return nil
	}
	return &telegram.InlineKeyboard{Rows: b.rows}
}
