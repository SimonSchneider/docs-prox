package config

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/SimonSchneider/docs-prox/providers/environment"

	"github.com/SimonSchneider/docs-prox/openapi"
	"github.com/SimonSchneider/docs-prox/providers/file"
	"github.com/SimonSchneider/docs-prox/providers/kubernetes"
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
		return nil, err
	}
	defer file.Close()
	return Parse(file)
}

// Parse creates a config from a io.Reader
func Parse(r io.Reader) (*Config, error) {
	var c Config
	err := json.NewDecoder(r).Decode(&c)
	return &c, err
}

// BuildRepo builds a repo and APIStore
func (c *Config) BuildRepo(ctx context.Context) (openapi.Repository, openapi.SpecStore, error) {
	cachedRepo := openapi.NewCachedRepository()
	apiStore := openapi.Logging(cachedRepo)
	if conf := c.Providers.Environment; conf.Enabled {
		environment.Configure(apiStore, conf.Prefix)
	}
	if conf := c.Providers.File; conf.Enabled {
		err := file.Configure(ctx, apiStore, conf.Path, conf.Prefix)
		if err != nil {
			return nil, nil, err
		}
	}
	if conf := c.Providers.Kubernetes; conf.Enabled {
		err := kubernetes.Configure(ctx, apiStore)
		if err != nil {
			return nil, nil, err
		}
	}
	return openapi.Sorted(cachedRepo), apiStore, nil
}
