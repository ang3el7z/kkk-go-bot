package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ang3el7z/kkk-go-bot/internal/adguard"
	"github.com/ang3el7z/kkk-go-bot/internal/config"
	"github.com/ang3el7z/kkk-go-bot/internal/legacy"
	"github.com/ang3el7z/kkk-go-bot/internal/services"
	"github.com/ang3el7z/kkk-go-bot/internal/storage"
	"github.com/ang3el7z/kkk-go-bot/internal/telegram"
	"github.com/ang3el7z/kkk-go-bot/internal/usecase"
	"github.com/ang3el7z/kkk-go-bot/internal/wireguard"
	"github.com/ang3el7z/kkk-go-bot/internal/xray"
)

type Options struct {
	Smoke bool
}

func Run(ctx context.Context, cfg config.Config, opts Options) error {
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return err
	}
	repo, err := storage.OpenSQLite(cfg.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		return err
	}
	if err := legacy.NewImporter(cfg, repo).Import(ctx); err != nil {
		return err
	}
	registry := services.NewRegistry(repo, services.ComposeFile{Path: cfg.ComposePath}, services.DockerRuntime{})
	if err := registry.Refresh(ctx); err != nil {
		return err
	}

	if opts.Smoke {
		admins, _ := repo.ListAdmins(ctx)
		menuServices, _ := repo.MenuServices(ctx)
		fmt.Printf("db=%s admins=%d menu_services=%d\n", cfg.DBPath, len(admins), len(menuServices))
		return nil
	}
	if err := cfg.ValidateRuntime(); err != nil {
		return err
	}

	xrayManager := xray.NewManager(cfg, repo)
	bot := usecase.NewBot(repo, wireguard.NewManager(cfg, repo), xrayManager, adguard.NewManager(cfg, repo))
	go runXrayStatsLoop(ctx, xrayManager)
	client := telegram.NewAPIClient(cfg.TelegramToken)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/telegram/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var update telegram.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := handleUpdate(r.Context(), bot, client, update); err != nil {
			log.Printf("handle update: %v", err)
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	pacHandler := func(w http.ResponseWriter, r *http.Request) {
		handlePAC(r.Context(), xrayManager, w, r)
	}
	mux.HandleFunc("/pac", pacHandler)
	mux.HandleFunc("/pac/", pacHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/pac") {
			handlePAC(r.Context(), xrayManager, w, r)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("kkk-go-bot listening on %s", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func handlePAC(ctx context.Context, manager *xray.Manager, w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("s")
	if uuid == "" {
		uuid = r.URL.Query().Get("id")
	}
	typ := r.URL.Query().Get("t")
	if legacyUUID, legacyType := legacyPACParams(r.URL.Path); uuid == "" && legacyUUID != "" {
		uuid = legacyUUID
		if typ == "" {
			typ = legacyType
		}
	}
	if typ == "" && strings.HasSuffix(r.URL.Path, "/sub") {
		typ = "s"
	}
	baseURL := publicBaseURL(r)
	if redirect := r.URL.Query().Get("r"); redirect != "" {
		if redirect == "w" {
			name, body, err := manager.WindowsZip(ctx, uuid, baseURL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
			_, _ = w.Write(body)
			return
		}
		location, err := manager.ImportRedirect(ctx, uuid, typ, redirect, baseURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Redirect(w, r, location, http.StatusFound)
		return
	}
	contentType, body, err := manager.Subscription(ctx, uuid, typ)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write([]byte(body))
}

func publicBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

var phpSerializedPair = regexp.MustCompile(`s:\d+:"([^"]+)";s:\d+:"([^"]*)";`)

func legacyPACParams(path string) (string, string) {
	segment := path[strings.LastIndex(path, "/")+1:]
	if segment == "" || segment == "sub" || strings.HasPrefix(segment, "pac") {
		return "", ""
	}
	data, err := base64.StdEncoding.DecodeString(segment)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(segment)
	}
	if err != nil {
		return "", ""
	}
	values := map[string]string{}
	for _, match := range phpSerializedPair.FindAllStringSubmatch(string(data), -1) {
		values[match[1]] = match[2]
	}
	return values["s"], values["t"]
}

func runXrayStatsLoop(ctx context.Context, manager *xray.Manager) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := manager.RefreshStats(ctx); err != nil {
				log.Printf("refresh xray stats: %v", err)
			}
		}
	}
}

func handleUpdate(ctx context.Context, bot *usecase.Bot, client telegram.Client, update telegram.Update) error {
	switch {
	case update.Message != nil:
		result, err := bot.HandleMessage(ctx, *update.Message)
		if err != nil {
			return err
		}
		return client.SendMessage(update.Message.Chat.ID, result.Text, result.Keyboard)
	case update.CallbackQuery != nil:
		result, err := bot.HandleCallback(ctx, *update.CallbackQuery)
		if err != nil {
			return err
		}
		if result.Keyboard != nil && update.CallbackQuery.Message != nil {
			if err := client.SendMessage(update.CallbackQuery.Message.Chat.ID, result.Text, result.Keyboard); err != nil {
				return err
			}
			return client.AnswerCallbackQuery(update.CallbackQuery.ID, "", false)
		}
		if result.Document != nil && update.CallbackQuery.Message != nil {
			if err := client.SendDocument(update.CallbackQuery.Message.Chat.ID, result.Document.Filename, result.Document.Content); err != nil {
				return err
			}
			return client.AnswerCallbackQuery(update.CallbackQuery.ID, "sent", false)
		}
		if result.Photo != nil && update.CallbackQuery.Message != nil {
			if err := client.SendPhoto(update.CallbackQuery.Message.Chat.ID, *result.Photo); err != nil {
				return err
			}
			return client.AnswerCallbackQuery(update.CallbackQuery.ID, "sent", false)
		}
		return client.AnswerCallbackQuery(update.CallbackQuery.ID, result.Text, result.ShowAlert)
	default:
		return nil
	}
}
