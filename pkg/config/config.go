package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/SimonSchneider/docs-prox/pkg/providers/environment"

	"github.com/SimonSchneider/docs-prox/pkg/openapi"
	"github.com/SimonSchneider/docs-prox/pkg/providers/file"
	"github.com/SimonSchneider/docs-prox/pkg/providers/kubernetes"
)

// Config is the json config file struct
type Config struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Providers struct {
		Environment struct {
			Enabled bool   `json:"enabled"`
			Prefix  string `json:"prefix"`
		} `json:"environment"`
		File struct {
			Enabled bool   `json:"enabled"`
			Path    string `json:"path"`
			Prefix  string `json:"prefix"`
			JSONExt string `json:"json-ext"`
			URLExt  string `json:"url-ext"`
		} `json:"file"`
		Kubernetes struct {
			Enabled bool `json:"enabled"`
		} `json:"kubernetes"`
	} `json:"providers"`
}

// ReadAndParseFile creates a config from a given filepath
func ReadAndParseFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file %s: %w", path, err)
	}
	defer file.Close()
	return Parse(file)
}

// Parse creates a config from a io.Reader
func Parse(r io.Reader) (*Config, error) {
	var c Config
	err := json.NewDecoder(r).Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}
	return &c, nil
}

// BuildRepo builds a repo and APIStore
func (c *Config) BuildRepo(ctx context.Context) (openapi.Repository, openapi.SpecStore, error) {
	cachedRepo := openapi.NewCachedRepository()
	apiStore := openapi.Logging(cachedRepo)
	if conf := c.Providers.Environment; conf.Enabled {
		environment.Configure(apiStore, conf.Prefix)
	}
	if conf := c.Providers.File; conf.Enabled {
		err := file.Configure(ctx, apiStore, conf.Path, conf.Prefix, conf.JSONExt, conf.URLExt)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to configure file provider with config %v: %w", conf, err)
		}
	}
	if conf := c.Providers.Kubernetes; conf.Enabled {
		err := kubernetes.Configure(ctx, apiStore)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to configure kubernetes provider with config %v: %w", conf, err)
		}
	}
	return cachedRepo, apiStore, nil
}
