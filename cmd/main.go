package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SimonSchneider/docs-prox/openapi"

	"github.com/SimonSchneider/docs-prox/config"
)

func main() {
	fmt.Println("loading configuration")
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "_config/config.json"
	}
	conf, err := config.ReadAndParseFile(path)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo, _, err := conf.BuildRepo(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("starting server")
	_, errChan := openapi.Serve(ctx, repo, conf.Host, conf.Port)
	select {
	case err := <-errChan:
		panic(err)
	case <-ctx.Done():
		return
	}
}
