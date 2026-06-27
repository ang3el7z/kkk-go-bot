package services

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"
)

type DockerRuntime struct {
	Socket string
}

func (d DockerRuntime) RunningServices(ctx context.Context) (map[string]bool, error) {
	socket := d.Socket
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	if _, err := os.Stat(socket); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json", nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var containers []struct {
		State  string            `json:"State"`
		Labels map[string]string `json:"Labels"`
	}
	if err := json.NewDecoder(res.Body).Decode(&containers); err != nil {
		return nil, err
	}
	running := map[string]bool{}
	for _, container := range containers {
		if container.State != "running" {
			continue
		}
		service := container.Labels["com.docker.compose.service"]
		if service != "" {
			running[service] = true
		}
	}
	return running, nil
}
