package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/SimonSchneider/docs-prox/pkg/openapi"

	"github.com/SimonSchneider/docs-prox/pkg/config"
)

func main() {
	fmt.Println("loading configuration")
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "_config/config.json"
	}
	conf, err := config.ReadAndParseFile(path)
	if err != nil {
		log.Fatalf("unable to parse config file %s: %v", path, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo, _, err := conf.BuildRepo(ctx)
	if err != nil {
		log.Fatalf("unable to build repo from config: %v", err)
	}
	fmt.Println("starting server")
	_, errChan := openapi.Serve(ctx, repo, conf.Host, conf.Port)
	select {
	case err := <-errChan:
		log.Fatalf("serve failed with: %v", err)
	case <-ctx.Done():
		return
	}
}
