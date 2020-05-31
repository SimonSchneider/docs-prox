package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/SimonSchneider/docs-prox/openapi"
	"github.com/SimonSchneider/docs-prox/providers/environment"
	"github.com/SimonSchneider/docs-prox/providers/file"
)

type config struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	Providers struct {
		Environment struct {
			Enabled bool               `json:"enabled"`
			Config  environment.Config `json:"config"`
		} `json:"environment"`
		File struct {
			Enabled bool        `json:"enabled"`
			Config  file.Config `json:"config"`
		} `json:"file"`
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

func (c *config) buildProviders() openapi.Repsitory {
	cachedSpec1 := openapi.NewStaticSpec("{\"hi\":\"hello\"}")
	cachedSpec2 := openapi.NewStaticSpec("{\"hi\":\"hello from 2\"}")
	cachedSpec3 := openapi.NewStaticSpec("{\"hi\":\"hello from 3\"}")
	staticRepo := openapi.NewStaticRepo(map[string]openapi.Spec{"cachedSpec1": cachedSpec1, "cachedSpec2": cachedSpec2})
	remoteSpec1 := openapi.NewRemoteSpec("https://petstore.swagger.io/v2/swagger.json")
	remoteRepo := openapi.NewStaticRepo(map[string]openapi.Spec{"remoteSpec1": remoteSpec1})
	staticRepo2 := openapi.NewStaticRepo(map[string]openapi.Spec{"cachedSpec1": cachedSpec1, "cachedSpec3": cachedSpec3})
	repos := make([]openapi.Repsitory, 0)
	repos = append(repos, staticRepo, remoteRepo, staticRepo2)
	if c.Providers.Environment.Enabled {
		repos = append(repos, c.Providers.Environment.Config.Build())
	}
	if c.Providers.File.Enabled {
		file, err := c.Providers.File.Config.Build()
		if err != nil {
			fmt.Println(err)
		}
		repos = append(repos, file)
	}
	return openapi.AllOf(repos...)
}
