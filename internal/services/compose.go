package services

import (
	"context"
	"os"

	"gopkg.in/yaml.v3"
)

type ComposeFile struct {
	Path string
}

func (c ComposeFile) EnabledServices(ctx context.Context) (map[string]bool, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(doc.Services))
	for name, service := range doc.Services {
		enabled[name] = service != nil
	}
	return enabled, nil
}
