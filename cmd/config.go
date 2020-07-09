package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/SimonSchneider/docs-prox/providers/environment"

	"github.com/SimonSchneider/docs-prox/openapi"
	"github.com/SimonSchneider/docs-prox/providers/file"
	"github.com/SimonSchneider/docs-prox/providers/kubernetes"
)

type config struct {
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

func parse(path string) (*config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var c config
	defer file.Close()
	err = json.NewDecoder(file).Decode(&c)
	return &c, err
}

func (c *config) buildProviders(ctx context.Context) openapi.Repository {
	cachedRepo := openapi.NewCachedRepository()
	cachedSpec1 := openapi.NewStaticSpec(parseString("{\"hi\":\"hello\"}"))
	cachedSpec2 := openapi.NewStaticSpec(parseString("{\"hi\":\"hello from 2\"}"))
	cachedSpec3 := openapi.NewStaticSpec(parseString("{\"hi\":\"hello from 3\"}"))
	cachedRepo.Put("cached", "cachedSpec1", cachedSpec1)
	cachedRepo.Put("cached", "cachedSpec2", cachedSpec2)
	remoteSpec1 := openapi.NewRemoteSpec("https://petstore.swagger.io/v2/swagger.json")
	cachedRepo.Put("cached", "remoteSpec1", remoteSpec1)
	cachedRepo.Put("cached", "cachedSpec3", cachedSpec3)
	if c.Providers.Environment.Enabled {
		environment.Configure(cachedRepo, c.Providers.Environment.Config.Prefix)
	}
	if c.Providers.File.Enabled {
		err := file.Configure(ctx, cachedRepo, c.Providers.File.Config.Path, c.Providers.File.Config.Prefix)
		if err != nil {
			fmt.Println(err)
		}
	}
	if c.Providers.Kubernetes.Enabled {
		kubernetes.Configure(ctx, cachedRepo)
	}
	return openapi.Sorted(cachedRepo)
}

func parseString(str string) interface{} {
	var res interface{}
	json.Unmarshal([]byte(str), &res)
	return res
}
