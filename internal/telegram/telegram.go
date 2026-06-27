package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

type Update struct {
	ID            int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	ID   int64  `json:"message_id"`
	From User   `json:"from"`
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type InlineKeyboard struct {
	Rows [][]InlineButton
}

type InlineButton struct {
	Text string
	Data string
	URL  string
}

type Document struct {
	Filename string
	Content  []byte
}

type Client interface {
	SendMessage(chatID int64, text string, keyboard *InlineKeyboard) error
	SendDocument(chatID int64, filename string, content []byte) error
	AnswerCallbackQuery(callbackID, text string, showAlert bool) error
}

type APIClient struct {
	token string
	http  *http.Client
}

func NewAPIClient(token string) *APIClient {
	return &APIClient{
		token: token,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) SendMessage(chatID int64, text string, keyboard *InlineKeyboard) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": keyboardRows(keyboard),
		}
	}
	return c.call("sendMessage", payload)
}

func (c *APIClient) AnswerCallbackQuery(callbackID, text string, showAlert bool) error {
	return c.call("answerCallbackQuery", map[string]any{
		"callback_query_id": callbackID,
		"text":              text,
		"show_alert":        showAlert,
	})
}

func (c *APIClient) SendDocument(chatID int64, filename string, content []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", c.token)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("telegram sendDocument failed: %s", res.Status)
	}
	return nil
}

func (c *APIClient) call(method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)
	res, err := c.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("telegram %s failed: %s", method, res.Status)
	}
	return nil
}

func keyboardRows(keyboard *InlineKeyboard) [][]map[string]string {
	rows := make([][]map[string]string, 0, len(keyboard.Rows))
	for _, row := range keyboard.Rows {
		out := make([]map[string]string, 0, len(row))
		for _, button := range row {
			item := map[string]string{"text": button.Text}
			if button.URL != "" {
				item["url"] = button.URL
			} else {
				item["callback_data"] = button.Data
			}
			out = append(out, item)
		}
		if len(out) > 0 {
			rows = append(rows, out)
		}
	}
	return rows
}
