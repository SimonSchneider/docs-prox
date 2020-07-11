package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/SimonSchneider/docs-prox/config"
	"github.com/SimonSchneider/docs-prox/openapi"
)

func TestCombiningDifferentProviders(t *testing.T) {
	httpSpecServer := runHTTPSpecServer()
	fileSpecServer, err := newFileSpecServer("swagger-", ".json")
	if err != nil {
		t.Fatal(err)
	}
	defer fileSpecServer.Close()
	os.Setenv("SWAGGER_TEST", httpSpecServer.Add("test"))
	os.Setenv("NOT_EXISTING", httpSpecServer.Add("notRegistered"))
	fileSpecServer.Add("test-file-not-found.json")
	fileSpecServer.Add("swagger-not-found2.txt")
	fileSpecServer.Add("swagger-found-file.json")
	specServer := AllOf(httpSpecServer, fileSpecServer)
	tests := []struct {
		name    string
		before  func()
		config  TmplConfig
		numKeys int
	}{
		{
			name:    "no providers should have no keys",
			config:  TmplConfig{},
			numKeys: 0,
		},
		{
			name:    "env provider can be configured",
			config:  TmplConfig{EnvPrefix: "SWAGGER_"},
			numKeys: 1,
		},
		{
			name:    "file provider can be configure",
			config:  TmplConfig{FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix},
			numKeys: 1,
		},
		{
			name:    "env and file provider can be configured",
			config:  TmplConfig{EnvPrefix: "SWAGGER_", FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix},
			numKeys: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := runOpenAPIServer(test.config)
			if err != nil {
				t.Fatal(err)
			}
			err = validate(client, test.numKeys, specServer)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFileServerMutateDuringRun(t *testing.T) {
	fileSpecServer, err := newFileSpecServer("swagger-", ".json")
	if err != nil {
		t.Fatal(err)
	}
	defer fileSpecServer.Close()
	fileSpecServer.Add("swagger-found-file-1.json")
	client, err := runOpenAPIServer(TmplConfig{FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix})
	if err != nil {
		t.Fatal(err)
	}
	err = validate(client, 1, fileSpecServer)
	if err != nil {
		t.Fatal(err)
	}
	fileSpecServer.Add("swagger-found-file-2.json")
	err = await(100*time.Millisecond, 5*time.Second, func() error {
		return validate(client, 2, fileSpecServer)
	})
	if err != nil {
		t.Fatal(err)
	}
	fileSpecServer.Add("not-found-file.txt")
	err = await(1*time.Second, 5*time.Second, func() error {
		return validate(client, 2, fileSpecServer)
	})
	if err != nil {
		t.Fatal(err)
	}
	fileSpecServer.Delete("swagger-found-file-1.json")
	err = await(100*time.Millisecond, 5*time.Second, func() error {
		return validate(client, 1, fileSpecServer)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runOpenAPIServer(tmplConfig TmplConfig) (*testClient, error) {
	testConfig, err := newTestConfig()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	str, err := testConfig.configWith(tmplConfig)
	if err != nil {
		return nil, err
	}
	conf, err := config.Parse(strings.NewReader(str))
	if err != nil {
		return nil, err
	}
	repo, _, err := conf.BuildRepo(ctx)
	if err != nil {
		return nil, err
	}
	listener, _ := openapi.Serve(ctx, repo, conf.Host, conf.Port)
	return newTestClient(listener.Addr()), nil
}

func validate(client *testClient, numKeys int, server SpecServer) error {
	keys, err := client.getKeys()
	if err != nil {
		return err
	}
	if len(keys) != numKeys {
		return fmt.Errorf("got an unexpected number of keys %d: %v", len(keys), keys)
	}
	for _, key := range keys {
		spec, err := client.getSpec(key.Path)
		if err != nil {
			return err
		}
		if servedSpec, ok := server.Get(key.Name); ok {
			if spec.ID != servedSpec.ID {
				return fmt.Errorf("got incorrect spec for %s", key.Name)
			}
		} else {
			return fmt.Errorf("got key that is not being served")
		}
	}
	return nil
}

type testClient struct {
	client *http.Client
	addr   string
}

func newTestClient(addr net.Addr) *testClient {
	return &testClient{
		client: &http.Client{},
		addr:   fmt.Sprintf("http://%s", addr.String()),
	}
}

func (t *testClient) getKeys() ([]openapi.KeyUrls, error) {
	res, err := t.client.Get(fmt.Sprintf("%s/docs/", t.addr))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response code %d", res.StatusCode)
	}
	var keys []openapi.KeyUrls
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&keys)
	return keys, err
}

func (t *testClient) getSpec(path string) (SpecResp, error) {
	res, err := t.client.Get(fmt.Sprintf("%s%s", t.addr, path))
	if err != nil {
		return SpecResp{}, err
	}
	if res.StatusCode != http.StatusOK {
		return SpecResp{}, fmt.Errorf("bad response code %d", res.StatusCode)
	}
	var spec SpecResp
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&spec)
	return spec, err
}

type SpecResp struct {
	ID string
}

type SpecServer interface {
	Get(key string) (SpecResp, bool)
}

type SpecServers []SpecServer

func (s SpecServers) Get(key string) (SpecResp, bool) {
	for _, v := range s {
		if res, ok := v.Get(key); ok {
			return res, ok
		}
	}
	return SpecResp{}, false
}

func AllOf(servers ...SpecServer) SpecServer {
	var s SpecServers
	s = servers
	return s
}

type httpSpecServer struct {
	addr      string
	mu        sync.RWMutex
	responses map[string]SpecResp
}

func runHTTPSpecServer() *httpSpecServer {
	var addr = "localhost:30000"
	server := &httpSpecServer{
		addr:      fmt.Sprintf("http://%s", addr),
		mu:        sync.RWMutex{},
		responses: make(map[string]SpecResp),
	}
	go server.run()
	return server
}

func (s *httpSpecServer) run() {
	http.ListenAndServe(strings.TrimPrefix(s.addr, "http://"), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Printf("got request for %s\n", r.URL.Path)
		key := strings.TrimPrefix(r.URL.Path, "/")
		s.mu.RLock()
		defer s.mu.RUnlock()
		if spec, ok := s.responses[key]; ok {
			json.NewEncoder(rw).Encode(spec)
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
}

func (s *httpSpecServer) Add(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[key] = randomSwaggerResp()
	return fmt.Sprintf("%s/%s", s.addr, key)
}

func (s *httpSpecServer) Get(key string) (SpecResp, bool) {
	spec, ok := s.responses[key]
	return spec, ok
}

func randomSwaggerResp() SpecResp {
	return SpecResp{ID: strconv.Itoa(rand.Int())}
}

type TmplConfig struct {
	EnvPrefix  string
	FilePath   string
	FilePrefix string
}

type testConfig struct {
	tmpl *template.Template
}

func newTestConfig() (*testConfig, error) {
	tpl, err := template.New("config").Parse(`
{
	"host": "",
	"port": 0,
	"providers": {
		{{- if (not (eq .EnvPrefix ""))}}
		"environment": {
			"enabled": true,
			"prefix": "{{ .EnvPrefix }}"
		},
		{{- end}}
		{{- if (not ( eq .FilePath "")) }}
		 "file": {
			"enabled": true,
			"path": "{{ .FilePath }}",
			"prefix": "{{ .FilePrefix }}"
        },
		{{- end}}
		"thisisignored": 2
	}
}
`)
	if err != nil {
		return nil, err
	}
	return &testConfig{tpl}, nil
}

func (t *testConfig) configWith(conf TmplConfig) (string, error) {
	var tpl bytes.Buffer
	if err := t.tmpl.Execute(&tpl, conf); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

type fileSpecServer struct {
	dir    string
	prefix string
	ext    string
	specs  map[string]SpecResp
}

func newFileSpecServer(prefix, ext string) (*fileSpecServer, error) {
	dir, err := ioutil.TempDir("", "swaggers")
	if err != nil {
		return nil, err
	}
	return &fileSpecServer{
		dir:    dir,
		prefix: prefix,
		ext:    ext,
		specs:  make(map[string]SpecResp, 0),
	}, nil
}

func (f *fileSpecServer) Close() error {
	filepath.Walk(f.dir, func(path string, info os.FileInfo, err error) error {
		return os.Remove(path)
	})
	os.Remove(f.dir)
	return nil
}

func (f *fileSpecServer) Get(key string) (SpecResp, bool) {
	s, ok := f.specs[key]
	return s, ok
}

func (f *fileSpecServer) Add(fileName string) error {
	resp := randomSwaggerResp()
	path := filepath.Join(f.dir, fileName)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(resp)
	if err != nil {
		return err
	}
	f.fileIsSpec(fileName, func(key string) {
		f.specs[key] = resp
	})
	return nil
}

func (f *fileSpecServer) Delete(fileName string) error {
	f.fileIsSpec(fileName, func(key string) {
		delete(f.specs, key)
	})
	return os.Remove(filepath.Join(f.dir, fileName))
}

func (f *fileSpecServer) fileIsSpec(fileName string, fun func(name string)) {
	if strings.HasPrefix(fileName, f.prefix) && strings.HasSuffix(fileName, f.ext) {
		fun(strings.TrimPrefix(strings.TrimSuffix(fileName, f.ext), f.prefix))
	}
}

func await(interval, atMost time.Duration, runner func() error) error {
	timeout := make(chan struct{})
	try := make(chan struct{})
	go func() {
		select {
		case <-time.After(atMost):
			close(timeout)
		case <-timeout:
			return
		}
	}()
	go func() {
		for {
			select {
			case <-timeout:
				close(try)
				return
			case <-time.After(interval):
				try <- struct{}{}
			}
		}
	}()
	errors := make(chan error)
	go func() {
		defer close(errors)
		for {
			select {
			case <-timeout:
				errors <- fmt.Errorf("timed out waiting for condition")
				return
			case <-try:
				err := runner()
				if err == nil {
					return
				}
				fmt.Printf("failed to try %v\n", err)
			}
		}
	}()
	return <-errors
}
