package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/swag"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	serve(9010, "https://petstore.swagger.io/v2/swagger.json")
}

func serve(port int, spec string) error {
	specDoc, err := loads.Spec(spec)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(specDoc.Spec(), "", "  ")
	if err != nil {
		return err
	}

	basePath := "/"

	listener, err := net.Listen("tcp4", net.JoinHostPort("", strconv.Itoa(port)))
	if err != nil {
		return err
	}
	sh, sp, err := swag.SplitHostPort(listener.Addr().String())
	if err != nil {
		return err
	}
	if sh == "0.0.0.0" {
		sh = "localhost"
	}

	visit := "http://localhost:9010/swagger/"
	u, err := url.Parse(visit)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Add("url", fmt.Sprintf("http://%s:%d%s", sh, sp, path.Join(basePath, "swagger.json")))
	u.RawQuery = q.Encode()
	visit = u.String()

	fs := http.FileServer(http.Dir("./dist"))
	r := mux.NewRouter()
	r.Handle("/swagger.json", handlers.CORS()(specHandler(b)))
	r.Handle("/redoc/", RedocHandler(RedocOpts{
		BasePath: basePath,
		SpecURL:  path.Join(basePath, "swagger.json"),
		Path:     "docs",
	}))
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", fs))

	errFuture := make(chan error)
	go func() {
		docServer := new(http.Server)
		docServer.SetKeepAlivesEnabled(true)
		docServer.Handler = r

		errFuture <- docServer.Serve(listener)
	}()

	log.Println("serving docs at", visit)
	return <-errFuture
}

func specHandler(b []byte) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		//#nosec
		_, _ = rw.Write(b)
	})
}

// RedocOpts configures the Redoc middlewares
type RedocOpts struct {
	// BasePath for the UI path, defaults to: /
	BasePath string
	// Path combines with BasePath for the full UI path, defaults to: docs
	Path string
	// SpecURL the url to find the spec for
	SpecURL string
	// RedocURL for the js that generates the redoc site, defaults to: https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js
	RedocURL string
	// Title for the documentation site, default to: API documentation
	Title string
}

// EnsureDefaults in case some options are missing
func (r *RedocOpts) EnsureDefaults() {
	if r.BasePath == "" {
		r.BasePath = "/"
	}
	if r.Path == "" {
		r.Path = "docs"
	}
	if r.SpecURL == "" {
		r.SpecURL = "/swagger.json"
	}
	if r.RedocURL == "" {
		r.RedocURL = redocLatest
	}
	if r.Title == "" {
		r.Title = "API documentation"
	}
}

// Redoc creates a middleware to serve a documentation site for a swagger spec.
// This allows for altering the spec before starting the http listener.
//
func RedocHandler(opts RedocOpts) http.Handler {
	opts.EnsureDefaults()

	tmpl := template.Must(template.New("redoc").Parse(redocTemplate))

	buf := bytes.NewBuffer(nil)
	_ = tmpl.Execute(buf, opts)
	b := buf.Bytes()

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(http.StatusOK)

		_, _ = rw.Write(b)
		return
	})
}

const (
	redocLatest   = "https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js"
	redocTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
		<!-- needed for adaptive design -->
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <!--
    ReDoc doesn't change outer page styles
    -->
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-url='{{ .SpecURL }}'></redoc>
    <script src="{{ .RedocURL }}"> </script>
  </body>
</html>
`
)
