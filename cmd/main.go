package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SimonSchneider/docs-prox/openapi"

	"github.com/SimonSchneider/docs-prox/config"
)

func main() {
	fmt.Println("loading configuration")
	//config, _ := parse("_config/config.json")
	conf, err := config.Parse("_config/config.json")
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	repo, _, err := conf.BuildRepo(ctx)
	if err != nil {
		cancel()
		panic(err)
	}
	fmt.Println("starting server")
	go func() {
		<-time.After(1 * time.Second)
		cancel()
	}()
	err = openapi.Serve(ctx, repo, conf.Host, conf.Port)
	if err != nil {
		cancel()
		panic(err)
	}
}
