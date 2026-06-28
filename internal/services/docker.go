package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type DockerRuntime struct {
	Socket      string
	ProjectName string
}

func (d DockerRuntime) RunningServices(ctx context.Context) (map[string]bool, error) {
	socket := d.Socket
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	if _, err := os.Stat(socket); err != nil {
		return nil, err
	}
	client := d.client(socket)
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
		if !d.matchesProject(container.Labels) {
			continue
		}
		service := container.Labels["com.docker.compose.service"]
		if service != "" {
			running[service] = true
		}
	}
	return running, nil
}

func (d DockerRuntime) SetServiceRunning(ctx context.Context, service string, running bool) error {
	container, err := d.findServiceContainer(ctx, service)
	if err != nil {
		return err
	}
	if running && container.State == "running" {
		return nil
	}
	if !running && container.State != "running" {
		return nil
	}
	action := "stop?t=5"
	if running {
		action = "start"
	}
	return d.containerAction(ctx, container.ID, action)
}

type dockerContainer struct {
	ID     string            `json:"Id"`
	State  string            `json:"State"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
}

func (d DockerRuntime) findServiceContainer(ctx context.Context, service string) (dockerContainer, error) {
	socket := d.Socket
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	if _, err := os.Stat(socket); err != nil {
		return dockerContainer{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json?all=1", nil)
	if err != nil {
		return dockerContainer{}, err
	}
	res, err := d.client(socket).Do(req)
	if err != nil {
		return dockerContainer{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return dockerContainer{}, fmt.Errorf("docker containers list failed: %s %s", res.Status, strings.TrimSpace(string(body)))
	}
	var containers []dockerContainer
	if err := json.NewDecoder(res.Body).Decode(&containers); err != nil {
		return dockerContainer{}, err
	}
	for _, container := range containers {
		if !d.matchesProject(container.Labels) {
			continue
		}
		if container.Labels["com.docker.compose.service"] == service {
			return container, nil
		}
	}
	return dockerContainer{}, fmt.Errorf("container for service %q not found", service)
}

func (d DockerRuntime) containerAction(ctx context.Context, id, action string) error {
	socket := d.Socket
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://docker/containers/"+id+"/"+action, nil)
	if err != nil {
		return err
	}
	res, err := d.client(socket).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if (res.StatusCode >= 200 && res.StatusCode <= 299) || res.StatusCode == http.StatusNotModified {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	return fmt.Errorf("docker container action failed: %s %s", res.Status, strings.TrimSpace(string(body)))
}

func (d DockerRuntime) client(socket string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
	return &http.Client{Transport: transport, Timeout: 5 * time.Second}
}

func (d DockerRuntime) matchesProject(labels map[string]string) bool {
	if d.ProjectName == "" {
		return true
	}
	return labels["com.docker.compose.project"] == d.ProjectName
}
