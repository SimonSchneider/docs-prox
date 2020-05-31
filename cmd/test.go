package main

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	o "github.com/SimonSchneider/docs-prox/openapi"
	env "github.com/SimonSchneider/docs-prox/providers/environment"
	file "github.com/SimonSchneider/docs-prox/providers/localfile"
)

func main() {
	cachedSpec1 := o.NewStaticSpec("{\"hi\":\"hello\"}")
	cachedSpec2 := o.NewStaticSpec("{\"hi\":\"hello from 2\"}")
	cachedSpec3 := o.NewStaticSpec("{\"hi\":\"hello from 3\"}")
	staticRepo := o.NewStaticRepo(map[string]o.Spec{"cachedSpec1": cachedSpec1, "cachedSpec2": cachedSpec2})
	remoteSpec1 := o.NewRemoteSpec("https://petstore.swagger.io/v2/swagger.json")
	remoteRepo := o.NewStaticRepo(map[string]o.Spec{"remoteSpec1": remoteSpec1})
	staticRepo2 := o.NewStaticRepo(map[string]o.Spec{"cachedSpec1": cachedSpec1, "cachedSpec3": cachedSpec3})
	envRepo := env.NewEnvironmentRepsitory("SWAGGER_")
	fileRepo, _ := file.NewFileRepsitory("./conf", "swagger_")
	fullRepo := o.AllOf(staticRepo, remoteRepo, staticRepo2, envRepo, fileRepo)
	router(fullRepo)
}

func router(repo o.Repsitory) error {
	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("./dist"))
	r.Handle("/docs/", handlers.CORS()(keyHandler(repo)))
	r.Handle("/docs/{key}", handlers.CORS()(docsHandler(repo)))
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", fs))

	listener, err := net.Listen("tcp4", net.JoinHostPort("", strconv.Itoa(8080)))
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

func keyHandler(repo o.Repsitory) http.Handler {
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

func docsHandler(repo o.Repsitory) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		rw.Header().Set("Content-Type", "application/json")
		key := vars["key"]
		spec, err := repo.Spec(key)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		raw, err := spec.JsonSpec()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		bytes, err := json.Marshal(raw)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(bytes)
	})
}
