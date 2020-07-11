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
	conf, err := config.ReadAndParseFile("_config/config.json")
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
	go func() {
		<-time.After(1 * time.Second)
		//cancel()
	}()
	_, errChan := openapi.Serve(ctx, repo, conf.Host, conf.Port)
	select {
	case err := <-errChan:
		panic(err)
	case <-ctx.Done():
		return
	}
}
