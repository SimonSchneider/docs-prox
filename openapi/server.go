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

func Serve(ctx context.Context, repo Repository, host string, port int) error {
	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("./dist"))
	for _, fun := range []RepoHandlerFunc{keyHandler, docsHandler} {
		path, handler := fun(repo)
		r.Handle(fmt.Sprintf("/docs%s", path), handler)
	}
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", fs))

	listener, err := net.Listen("tcp4", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	errFuture := make(chan error)
	docServer := new(http.Server)
	shutdown := func() {
		deadline, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		fmt.Printf("shutting down server gracefully with a 10 second timeout\n")
		docServer.Shutdown(deadline)
	}
	defer shutdown()
	docServer.Handler = handlers.CORS()(r)
	go func() {
		defer close(errFuture)
		err := docServer.Serve(listener)
		errFuture <- err
	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errFuture:
		return err
	}
}

type RepoHandlerFunc func(repository Repository) (string, http.Handler)

func keyHandler(repo Repository) (string, http.Handler) {
	return "/", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		keys := repo.Keys()
		prep := make([]keyUrls, 0, len(keys))
		for _, k := range keys {
			prep = append(prep, keyUrls{Name: k, Path: r.URL.Path + k})
		}
		err := json.NewEncoder(rw).Encode(prep)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	})
}

type keyUrls struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func docsHandler(repo Repository) (string, http.Handler) {
	return "/{key}", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
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
