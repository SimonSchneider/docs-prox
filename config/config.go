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

type Config struct {
	Host      string        `json:"host"`
	Port      int           `json:"port"`
	Providers ProvidersConf `json:"providers"`
}

type ProvidersConf struct {
	Environment EnvironmentConf `json:"environment"`
	File        FileConf        `json:"file"`
	Kubernetes  KubernetesConf  `json:"kubernetes"`
}

type ProviderConf struct {
	Enabled bool `json:"enabled"`
}

type EnvironmentConf struct {
	ProviderConf
	Prefix string `json:"prefix"`
}

type FileConf struct {
	ProviderConf
	Path   string `json:"path"`
	Prefix string `json:"prefix"`
}

type KubernetesConf struct {
	ProviderConf
}

func ReadAndParseFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return Parse(file)
}

func Parse(r io.Reader) (*Config, error) {
	var c Config
	err := json.NewDecoder(r).Decode(&c)
	return &c, err
}

func (c *Config) BuildRepo(ctx context.Context) (openapi.Repository, openapi.ApiStore, error) {
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
