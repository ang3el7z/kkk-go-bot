package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ang3el7z/kkk-go-bot/internal/storage"
)

type smallServiceDefinition struct {
	Name  string
	Title string
	Keys  []string
}

var smallServiceDefinitions = map[string]smallServiceDefinition{
	"tg": {
		Name:  "tg",
		Title: "MTProto",
		Keys:  []string{"legacy.mtprotodomain", "legacy.mtprotosecret"},
	},
	"ss": {
		Name:  "ss",
		Title: "Shadowsocks",
		Keys:  []string{"legacy.ssserver.json"},
	},
	"proxy": {
		Name:  "proxy",
		Title: "SS Local Proxy",
		Keys:  []string{"legacy.sslocal.json"},
	},
	"oc": {
		Name:  "oc",
		Title: "OpenConnect",
		Keys:  []string{"legacy.ocserv.conf", "legacy.ocserv.passwd"},
	},
	"np": {
		Name:  "np",
		Title: "NaiveProxy",
		Keys:  []string{"legacy.caddyfile"},
	},
	"hy": {
		Name:  "hy",
		Title: "Hysteria",
		Keys:  []string{"legacy.hysteria.yaml"},
	},
	"dnstt": {
		Name:  "dnstt",
		Title: "DNSTT",
		Keys:  []string{"legacy.dnstt.server_pub", "legacy.dnstt.server_key"},
	},
	"wp": {
		Name:  "wp",
		Title: "Warp",
		Keys:  []string{"legacy.pac"},
	},
}

func (b *Bot) smallServiceMenu(ctx context.Context, name string) (CallbackResult, bool, error) {
	def, ok := smallServiceDefinitions[name]
	if !ok {
		return CallbackResult{}, false, nil
	}
	lines := []string{def.Title}
	for _, key := range def.Keys {
		setting, found, err := b.repo.GetSetting(ctx, key)
		if err != nil {
			return CallbackResult{}, true, err
		}
		lines = append(lines, smallSettingLine(key, setting, found))
	}
	keyboard := NewMenuBuilder(1)
	keyboard.Add("Back", "service:menu")
	return CallbackResult{Text: strings.Join(lines, "\n"), Keyboard: keyboard.Build()}, true, nil
}

func smallSettingLine(key string, setting storage.Setting, found bool) string {
	label := strings.TrimPrefix(key, "legacy.")
	if !found {
		return label + ": missing"
	}
	if setting.Secret {
		if setting.ValueJSON == "" || setting.ValueJSON == `""` {
			return label + ": secret empty"
		}
		return label + ": secret present"
	}
	var value any
	if err := json.Unmarshal([]byte(setting.ValueJSON), &value); err == nil {
		return label + ": " + summarizeValue(value)
	}
	return label + ": present"
}

func summarizeValue(value any) string {
	switch typed := value.(type) {
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return "empty"
		}
		if len(typed) > 72 {
			return typed[:72] + "..."
		}
		return typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 6 {
			keys = keys[:6]
		}
		return fmt.Sprintf("json keys=%s", strings.Join(keys, ","))
	case []any:
		return fmt.Sprintf("list items=%d", len(typed))
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func smallServiceNames() []string {
	names := make([]string, 0, len(smallServiceDefinitions))
	for name := range smallServiceDefinitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
