package xray

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m *Manager) RefreshStats(ctx context.Context) error {
	clients, err := m.repo.ListClients(ctx, "xray")
	if err != nil {
		return err
	}
	stats := m.readStats()
	if stats == nil {
		stats = xrayStats{}
	}
	users, _ := stats["users"].(map[string]any)
	if users == nil {
		users = map[string]any{}
		stats["users"] = users
	}
	for idx, client := range clients {
		down, _ := dockerXrayStat(ctx, "user>>>"+client.Name+">>>traffic>>>downlink")
		up, _ := dockerXrayStat(ctx, "user>>>"+client.Name+">>>traffic>>>uplink")
		user, _ := users[itoa(idx)].(map[string]any)
		if user == nil {
			user = map[string]any{}
			users[itoa(idx)] = user
		}
		setStatCounters(user, down, up)
	}
	body, err := json.MarshalIndent(stats, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.cfg.ConfigDir, "xray.stats"), body, 0o644)
}

func setStatCounters(values map[string]any, down, up int64) {
	session := map[string]any{"download": down, "upload": up}
	global, _ := values["global"].(map[string]any)
	if global == nil {
		global = map[string]any{}
	}
	global["download"] = nestedInt64(values, "global", "download") + down
	global["upload"] = nestedInt64(values, "global", "upload") + up
	values["session"] = session
	values["global"] = global
}

func dockerXrayStat(ctx context.Context, name string) (int64, error) {
	containerID, err := dockerContainerID(ctx, "xr")
	if err != nil {
		return 0, err
	}
	execID, err := dockerCreateExec(ctx, containerID, []string{"xray", "api", "stats", "--server=127.0.0.1:8080", "-name", name, "-reset"})
	if err != nil {
		return 0, err
	}
	out, err := dockerStartExec(ctx, execID)
	if err != nil {
		return 0, err
	}
	var payload struct {
		Stat struct {
			Value int64 `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return 0, err
	}
	return payload.Stat.Value, nil
}

func dockerClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", "/var/run/docker.sock")
			},
		},
	}
}

func dockerContainerID(ctx context.Context, service string) (string, error) {
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json", nil)
	if err != nil {
		return "", err
	}
	res, err := dockerClient().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var containers []struct {
		ID     string            `json:"Id"`
		Labels map[string]string `json:"Labels"`
	}
	if err := json.NewDecoder(res.Body).Decode(&containers); err != nil {
		return "", err
	}
	for _, container := range containers {
		if container.Labels["com.docker.compose.service"] == service {
			return container.ID, nil
		}
	}
	return "", errors.New("container not found")
}

func dockerCreateExec(ctx context.Context, containerID string, cmd []string) (string, error) {
	body, _ := json.Marshal(map[string]any{"AttachStdout": true, "AttachStderr": true, "Cmd": cmd})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://docker/containers/"+containerID+"/exec", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := dockerClient().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", errors.New("empty exec id")
	}
	return out.ID, nil
}

func dockerStartExec(ctx context.Context, execID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://docker/exec/"+execID+"/start", strings.NewReader(`{"Detach":false,"Tty":false}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := dockerClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return dockerDemux(data), nil
}

func dockerDemux(data []byte) []byte {
	var out []byte
	for len(data) >= 8 {
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		data = data[8:]
		if size <= 0 || size > len(data) {
			break
		}
		out = append(out, data[:size]...)
		data = data[size:]
	}
	if len(out) == 0 {
		return data
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var out []byte
	for v > 0 {
		out = append([]byte{byte('0' + v%10)}, out...)
		v /= 10
	}
	return string(out)
}
