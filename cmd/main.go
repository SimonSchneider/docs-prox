package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/SimonSchneider/docs-prox/openapi"
)

func main() {
	fmt.Println("loading configuration")
	config, _ := parse("_config/config.json")
	fmt.Println("setting up providers")
	ctx, cancel := context.WithCancel(context.Background())
	repo := config.buildProviders(ctx)
	fmt.Println("starting server")
	go func() {
		<-time.After(10 * time.Second)
		cancel()
	}()
	router(repo, config.Host, config.Port)
}

func router(repo openapi.Repository, host string, port int) error {
	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("./dist"))
	r.Handle("/docs/", handlers.CORS()(keyHandler(repo)))
	r.Handle("/docs/{key}", handlers.CORS()(docsHandler(repo)))
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", fs))

	listener, err := net.Listen("tcp4", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	errFuture := make(chan error)
	go func() {
		docServer := new(http.Server)
		docServer.SetKeepAlivesEnabled(true)
		docServer.Handler = r

		errFuture <- docServer.Serve(listener)
	}()

	return <-errFuture
}

func keyHandler(repo openapi.Repository) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		keys := repo.Keys()
		prep := make([]keyUrls, 0, len(keys))
		for _, k := range keys {
			prep = append(prep, keyUrls{Name: k, Path: r.URL.Path + k})
		}
		bytes, err := json.Marshal(prep)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(bytes)
	})
}

type keyUrls struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func docsHandler(repo openapi.Repository) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		rw.Header().Set("Content-Type", "application/json")
		key := vars["key"]
		spec, err := repo.Spec(key)
		if err != nil {
			fmt.Println(err)
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		raw, err := spec.JSONSpec()
		if err != nil {
			fmt.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		bytes, err := json.Marshal(raw)
		if err != nil {
			fmt.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(bytes)
	})
}
