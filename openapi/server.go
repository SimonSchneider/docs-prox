package openapi

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
)

// Serve starts a server that serves the repo
func Serve(ctx context.Context, repo Repository, host string, port int) (net.Listener, <-chan error) {
	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("./dist"))
	for _, fun := range []repoHandlerFunc{keyHandler, docsHandler} {
		path, handler := fun(repo)
		r.Handle(fmt.Sprintf("/docs%s", path), handler)
	}
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	listener, err := net.Listen("tcp4", net.JoinHostPort(host, strconv.Itoa(port)))
	errFuture := make(chan error)
	if err != nil {
		defer close(errFuture)
		errFuture <- err
		return nil, errFuture
	}
	docServer := new(http.Server)
	docServer.Handler = handlers.CORS()(r)
	go func() {
		defer close(errFuture)
		err := docServer.Serve(listener)
		errFuture <- err
	}()
	go func() {
		<-ctx.Done()
		deadline, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		defer cancel()
		fmt.Printf("shutting down server gracefully with a 10 second timeout\n")
		docServer.Shutdown(deadline)
	}()
	return listener, errFuture
}

type repoHandlerFunc func(repository Repository) (string, http.Handler)

func keyHandler(repo Repository) (string, http.Handler) {
	return "/", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		keys := repo.Keys()
		prep := make([]KeyUrls, 0, len(keys))
		for _, k := range keys {
			prep = append(prep, KeyUrls{Id: k.Key, Name: k.Name, Path: r.URL.Path + k.Key})
		}
		err := json.NewEncoder(rw).Encode(prep)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// KeyUrls is returned in the Keys endpoint
type KeyUrls struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

func docsHandler(repo Repository) (string, http.Handler) {
	return "/{keyId}", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		rw.Header().Set("Content-Type", "application/json")
		keyId := vars["keyId"]
		spec, err := repo.Spec(keyId)
		if err != nil {
			fmt.Printf("unable to get keyId %s: %v\n", keyId, err)
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		bytes, err := spec.Get()
		if err != nil {
			fmt.Printf("unable to retrieve spec %s: %v\n", keyId, err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(bytes)
	})
}
