package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMessagePayloadAndKeyboard(t *testing.T) {
	var path string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("bad content-type: %s", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	client := NewAPIClientWithHTTP("token", server.URL, server.Client())
	err := client.SendMessage(42, "hello", &InlineKeyboard{Rows: [][]InlineButton{{
		{Text: "A", Data: "a:1"},
		{Text: "Docs", URL: "https://example.com"},
		{Text: "App", WebApp: "https://example.com/app"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/bottoken/sendMessage" {
		t.Fatalf("bad path: %s", path)
	}
	if payload["chat_id"] != float64(42) || payload["text"] != "hello" || payload["parse_mode"] != "HTML" {
		t.Fatalf("bad payload: %+v", payload)
	}
	markup, _ := payload["reply_markup"].(map[string]any)
	rows, _ := markup["inline_keyboard"].([]any)
	if len(rows) != 1 {
		t.Fatalf("bad keyboard: %+v", markup)
	}
	firstRow, _ := rows[0].([]any)
	webAppButton, _ := firstRow[2].(map[string]any)
	if _, ok := webAppButton["web_app"].(map[string]any); !ok {
		t.Fatalf("missing web_app button: %+v", webAppButton)
	}
}

func TestSendDocumentMultipartAndAPIError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Fatalf("bad content-type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "client.conf") || !strings.Contains(string(body), "peer") {
				t.Fatalf("bad multipart body: %s", body)
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":false,"description":"bad token"}`))
	}))
	defer server.Close()

	client := NewAPIClientWithHTTP("token", server.URL, server.Client())
	if err := client.SendDocument(42, "client.conf", []byte("peer")); err != nil {
		t.Fatal(err)
	}
	err := client.AnswerCallbackQuery("cb", "nope", true)
	if err == nil || !strings.Contains(err.Error(), "bad token") {
		t.Fatalf("expected API error, got %v", err)
	}
}
