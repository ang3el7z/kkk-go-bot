package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	TelegramToken string
	HTTPAddr      string
	DataDir       string
	DBPath        string
	ComposePath   string
	ConfigDir     string
	CertsDir      string
	LogsDir       string
	LegacyPHPPath string
	PublicIP      string
	ProjectName   string
	WGAddress     string
	WG1Address    string
	WGPort        string
	WG1Port       string
	Domain        string
	ReloadWG      bool
}

func Load(envFile string) (Config, error) {
	if envFile != "" {
		if err := loadEnvFile(envFile); err != nil {
			return Config{}, err
		}
	}

	dataDir := getenv("DATA_DIR", "data")
	cfg := Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DataDir:       dataDir,
		DBPath:        getenv("DB_PATH", filepath.Join(dataDir, "bot.db")),
		ComposePath:   getenv("COMPOSE_PATH", "docker-compose.yml"),
		ConfigDir:     getenv("CONFIG_DIR", "config"),
		CertsDir:      getenv("CERTS_DIR", "certs"),
		LogsDir:       getenv("LOGS_DIR", "logs"),
		LegacyPHPPath: getenv("LEGACY_PHP_CONFIG", filepath.Join("app", "config.php")),
		PublicIP:      getenv("PUBLIC_IP", getenv("IP", firstNonLoopbackIP())),
		ProjectName:   getenv("PROJECT_NAME", "kkk-go-bot"),
		WGAddress:     getenv("WGADDRESS", "10.0.1.1/24"),
		WG1Address:    getenv("WG1ADDRESS", "10.0.3.1/24"),
		WGPort:        getenv("WGPORT", "51820"),
		WG1Port:       getenv("WG1PORT", "51821"),
		Domain:        os.Getenv("DOMAIN"),
		ReloadWG:      os.Getenv("WG_RELOAD") == "1",
	}

	if cfg.HTTPAddr == "" {
		return Config{}, errors.New("HTTP_ADDR is empty")
	}
	if cfg.DBPath == "" {
		return Config{}, errors.New("DB_PATH is empty")
	}
	return cfg, nil
}

func (c Config) ValidateRuntime() error {
	if c.TelegramToken == "" {
		return errors.New("TELEGRAM_BOT_TOKEN is required for Telegram runtime")
	}
	return nil
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open env file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonLoopbackIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip != nil {
				return ip.String()
			}
		}
	}
	return ""
}
