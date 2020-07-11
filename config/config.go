package config

import (
	"context"
	"encoding/json"
	"os"

	"github.com/SimonSchneider/docs-prox/providers/environment"

	"github.com/SimonSchneider/docs-prox/openapi"
	"github.com/SimonSchneider/docs-prox/providers/file"
	"github.com/SimonSchneider/docs-prox/providers/kubernetes"
)

type Config struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Providers struct {
		Environment struct {
			Enabled bool `json:"enabled"`
			Config  struct {
				Prefix string `json:"prefix"`
			} `json:"config"`
		} `json:"environment"`
		File struct {
			Enabled bool `json:"enabled"`
			Config  struct {
				Path   string `json:"path"`
				Prefix string `json:"prefix"`
			} `json:"config"`
		} `json:"file"`
		Kubernetes struct {
			Enabled bool `json:"enabled"`
		} `json:"kubernetes"`
	}
}

func Parse(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var c Config
	defer file.Close()
	err = json.NewDecoder(file).Decode(&c)
	return &c, err
}

func (c *Config) BuildRepo(ctx context.Context) (openapi.Repository, openapi.ApiStore, error) {
	cachedRepo := openapi.NewCachedRepository()
	staticSpecs := map[string]string{
		"cachedSpec1": "{\"hi\":\"hello\"}",
		"cachedSpec2": "{\"hi\":\"hello from 2\"}",
		"cachedSpec3": "{\"hi\":\"hello from 3\"}",
	}
	for n, v := range staticSpecs {
		spec, err := staticSpec(v)
		if err != nil {
			return nil, nil, err
		}
		cachedRepo.Put("cached", n, spec)
	}
	cachedRepo.Put("cached", "remoteSpec1", openapi.NewRemoteSpec("https://petstore.swagger.io/v2/swagger.json"))
	if c.Providers.Environment.Enabled {
		environment.Configure(cachedRepo, c.Providers.Environment.Config.Prefix)
	}
	if c.Providers.File.Enabled {
		conf := c.Providers.File.Config
		err := file.Configure(ctx, cachedRepo, conf.Path, conf.Prefix)
		if err != nil {
			return nil, nil, err
		}
	}
	if c.Providers.Kubernetes.Enabled {
		err := kubernetes.Configure(ctx, cachedRepo)
		if err != nil {
			return nil, nil, err
		}
	}
	return openapi.Sorted(cachedRepo), cachedRepo, nil
}

func staticSpec(str string) (openapi.Spec, error) {
	parsed, err := parseString(str)
	if err != nil {
		return nil, err
	}
	return openapi.NewStaticSpec(parsed), nil
}

func parseString(str string) (interface{}, error) {
	var res interface{}
	err := json.Unmarshal([]byte(str), &res)
	return res, err
}
