package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	"github.com/SimonSchneider/docs-prox/pkg/test/shell"

	"github.com/SimonSchneider/docs-prox/pkg/test/await"

	"github.com/SimonSchneider/docs-prox/pkg/config"
	"github.com/SimonSchneider/docs-prox/pkg/openapi"
)

func TestCombiningDifferentProviders(t *testing.T) {
	envPrefix := strconv.Itoa(rand.Int())
	httpSpecServer := runHTTPSpecServer()
	minikube, err := startMiniKube()
	check(t, err)
	defer minikube.Close()
	check(t, minikube.addConfigMap("swagger-test", map[string]string{
		"service-in-cm": httpSpecServer.Add("service-in-cm"),
	}))
	check(t, minikube.addService("test-service-in-kube"))
	fileSpecServer, err := newFileSpecServer("swagger-", ".json")
	check(t, err)
	defer fileSpecServer.Close()
	check(t, os.Setenv(envPrefix+"TEST", httpSpecServer.Add("test")))
	check(t, os.Setenv("NOT_EXISTING", httpSpecServer.Add("notRegistered")))
	check(t, fileSpecServer.AddJSONFile("test-file-not-found.json"))
	check(t, fileSpecServer.AddJSONFile("swagger-not-found2.txt"))
	check(t, fileSpecServer.AddJSONFile("swagger-found-file.json"))
	check(t, fileSpecServer.AddURLFile("swagger-found-url.url", map[string]string{"file-url-spec": httpSpecServer.Add("file-url-spec")}))
	specServer := AllOf(httpSpecServer, fileSpecServer)
	tests := []struct {
		name            string
		before          func()
		config          TmplConfig
		numKeys         int
		unreachableKeys []string
	}{
		{
			name:            "no providers should have no keys",
			config:          TmplConfig{},
			numKeys:         0,
			unreachableKeys: []string{},
		},
		{
			name:            "env provider can be configured",
			config:          TmplConfig{EnvPrefix: envPrefix},
			numKeys:         1,
			unreachableKeys: []string{},
		},
		{
			name:            "file provider can be configure",
			config:          TmplConfig{FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix},
			numKeys:         2,
			unreachableKeys: []string{},
		},
		{
			name:            "kubernetes provider can be configured",
			config:          TmplConfig{KubeEnabled: true},
			numKeys:         2,
			unreachableKeys: []string{"test-service-in-kube"},
		},
		{
			name:            "all providers can be configured",
			config:          TmplConfig{EnvPrefix: envPrefix, FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix, KubeEnabled: true},
			numKeys:         5,
			unreachableKeys: []string{"test-service-in-kube"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := runOpenAPIServer(test.config)
			check(t, err)
			check(t, await.AtMost(5*time.Second).That(func() error {
				return validate(client, test.numKeys, specServer, test.unreachableKeys...)
			}))
		})
	}
}

func TestFileServerMutateJsonDuringRun(t *testing.T) {
	fileSpecServer, err := newFileSpecServer("swagger-", ".json")
	check(t, err)
	defer fileSpecServer.Close()
	check(t, fileSpecServer.AddJSONFile("swagger-found-file-1.json"))
	client, err := runOpenAPIServer(TmplConfig{FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix})
	check(t, err)
	check(t, validate(client, 1, fileSpecServer))
	check(t, fileSpecServer.AddJSONFile("swagger-found-file-2.json"))
	check(t, await.That(func() error {
		return validate(client, 2, fileSpecServer)
	}))
	check(t, fileSpecServer.AddJSONFile("not-found-file.txt"))
	check(t, await.That(func() error {
		return validate(client, 2, fileSpecServer)
	}))
	check(t, fileSpecServer.Delete("swagger-found-file-1.json"))
	check(t, await.That(func() error {
		return validate(client, 1, fileSpecServer)
	}))
}

func TestFileServerMutateUrlDuringRun(t *testing.T) {
	fileSpecServer, err := newFileSpecServer("swagger-", ".json")
	check(t, err)
	defer fileSpecServer.Close()
	client, err := runOpenAPIServer(TmplConfig{FilePath: fileSpecServer.dir, FilePrefix: fileSpecServer.prefix})
	verifyKeys := func(expectedKeys ...string) error {
		keys, err := client.getKeys()
		check(t, err)
		if len(keys) != len(expectedKeys) {
			return fmt.Errorf("unexpected number of keys %d expected %d", len(keys), len(expectedKeys))
		}
		for _, k := range keys {
			foundKey := false
			for _, fk := range expectedKeys {
				if fk == k.Name {
					foundKey = true
				}
			}
			if !foundKey {
				return fmt.Errorf("couldn't find key %v in expected %v", k, expectedKeys)
			}
		}
		return nil
	}
	file1, key1, url1 := "swagger-found-file-1.url", "key1", "url1"
	check(t, err)
	check(t, fileSpecServer.AddURLFile(file1, map[string]string{key1: url1}))
	check(t, await.That(func() error { return verifyKeys(key1) }))
	file2, key2, url2 := "swagger-found-file-2.url", "key2", "url2"
	check(t, fileSpecServer.AddURLFile(file2, map[string]string{key2: url2}))
	check(t, await.That(func() error { return verifyKeys(key1, key2) }))
	check(t, fileSpecServer.Delete("swagger-found-file-1.url"))
	check(t, await.That(func() error { return verifyKeys(key2) }))
	check(t, fileSpecServer.OpenAndAppendToFile(file2, func(w io.Writer) error {
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		_, err := writer.WriteString("this is incorrectly encoded\n")
		return err
	}))
	nCheck("after appending bad", t, await.That(func() error { return verifyKeys(key2) }))
	check(t, fileSpecServer.OpenAndAppendToFile(file2, func(w io.Writer) error {
		return writeMap(w, map[string]string{key1: url1})
	}))
	nCheck("after appending good", t, await.That(func() error { return verifyKeys(key1, key2) }))
}

func Test404OnMissingKey(t *testing.T) {
	c, err := runOpenAPIServer(TmplConfig{})
	check(t, err)
	get, err := c.get(fmt.Sprintf("/docs/non-existing-key"))
	check(t, err)
	if get.StatusCode != http.StatusNotFound {
		t.Errorf("found key that should not be found %s", get.Status)
	}
}

func Test500OnBrokenUpstream(t *testing.T) {
	envPrefix := "SWAGGER_"
	check(t, os.Setenv(envPrefix+"TEST", "http://localhost:8080/does/not/exist"))
	c, err := runOpenAPIServer(TmplConfig{EnvPrefix: envPrefix})
	check(t, err)
	keys, err := c.getKeys()
	check(t, err)
	if len(keys) != 1 {
		t.Errorf("unexpected length of keys %v", keys)
	}
	r, err := c.get(keys[0].Path)
	check(t, err)
	if r.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 got %d %s", r.StatusCode, r.Status)
	}
}

func runOpenAPIServer(tmplConfig TmplConfig) (*testClient, error) {
	testConfig, err := newTestConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to create new config template: %w", err)
	}
	ctx := context.Background()
	path, err := testConfig.configWith(tmplConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to template config: %w", err)
	}
	conf, err := config.ReadAndParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read and parse config: %w", err)
	}
	repo, _, err := conf.BuildRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to build repositories: %w", err)
	}
	listener, _ := openapi.Serve(ctx, repo, conf.Host, conf.Port)
	return newTestClient(listener.Addr()), nil
}

func validate(client *testClient, numKeys int, server SpecServer, unreachableKeys ...string) error {
	keys, err := client.getKeys()
	if err != nil {
		return fmt.Errorf("unable to retrieve keys: %w", err)
	}
	if len(keys) != numKeys {
		return fmt.Errorf("got an unexpected number of keys %d: %v", len(keys), keys)
	}
	for _, key := range keys {
		if contains(unreachableKeys, key.Name) {
			continue
		}
		spec, err := client.getSpec(key.Path)
		if err != nil {
			return fmt.Errorf("unable to retrieve spec: %w", err)
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

func contains(arr []string, key string) bool {
	for _, item := range arr {
		if item == key {
			return true
		}
	}
	return false
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
	res, err := t.get("/docs/")
	if err != nil {
		return nil, fmt.Errorf("unable to do keys request: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response code %d", res.StatusCode)
	}
	var keys []openapi.KeyUrls
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&keys)
	if err != nil {
		return nil, fmt.Errorf("unable to decode keys reponse: %w", err)
	}
	return keys, nil
}

func (t *testClient) get(path string) (*http.Response, error) {
	return t.client.Get(fmt.Sprintf("%s%s", t.addr, path))
}

func (t *testClient) getSpec(path string) (SpecResp, error) {
	res, err := t.get(path)
	if err != nil {
		return SpecResp{}, fmt.Errorf("unable to do spec request: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return SpecResp{}, fmt.Errorf("bad response code %d", res.StatusCode)
	}
	var spec SpecResp
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&spec)
	if err != nil {
		return SpecResp{}, fmt.Errorf("unable to decode spec response: %w", err)
	}
	return spec, err
}

type SpecResp struct {
	ID string
}

func randomSwaggerResp() SpecResp {
	return SpecResp{ID: strconv.Itoa(rand.Int())}
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

type fileSpecServer struct {
	dir    string
	prefix string
	ext    string
	specs  map[string]SpecResp
}

func newFileSpecServer(prefix, ext string) (*fileSpecServer, error) {
	dir, err := ioutil.TempDir("", "swaggers")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary directory: %w", err)
	}
	return &fileSpecServer{
		dir:    dir,
		prefix: prefix,
		ext:    ext,
		specs:  make(map[string]SpecResp, 0),
	}, nil
}

func (f *fileSpecServer) Close() error {
	err := filepath.Walk(f.dir, func(path string, info os.FileInfo, err error) error {
		return os.Remove(path)
	})
	if err != nil {
		return fmt.Errorf("unable to clean up files: %w", err)
	}
	err = os.Remove(f.dir)
	if err != nil {
		return fmt.Errorf("unable to clean up dir: %w", err)
	}
	return nil
}

func (f *fileSpecServer) Get(key string) (SpecResp, bool) {
	s, ok := f.specs[key]
	return s, ok
}

func (f *fileSpecServer) CreateAndWriteToFile(fileName string, writeTo func(w io.Writer) error) error {
	path := filepath.Join(f.dir, fileName)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %w", path, err)
	}
	defer file.Close()
	if err := writeTo(file); err != nil {
		return fmt.Errorf("error writing to file %s, %w", path, err)
	}
	return nil
}

func (f *fileSpecServer) OpenAndAppendToFile(filename string, writeTo func(w io.Writer) error) error {
	path := filepath.Join(f.dir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %w", path, err)
	}
	defer file.Close()
	if err := writeTo(file); err != nil {
		return fmt.Errorf("error writing to file %s, %w", path, err)
	}
	return nil
}

func (f *fileSpecServer) AddJSONFile(fileName string) error {
	resp := randomSwaggerResp()
	err := f.CreateAndWriteToFile(fileName, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(resp)
	})
	if err != nil {
		return err
	}
	f.fileIsSpec(fileName, func(key string) {
		f.specs[key] = resp
	})
	return nil
}

func (f *fileSpecServer) AddURLFile(fileName string, specs map[string]string) error {
	return f.CreateAndWriteToFile(fileName, func(w io.Writer) error {
		return writeMap(w, specs)
	})
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

func writeMap(w io.Writer, toWrite map[string]string) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	for key, spec := range toWrite {
		_, err := writer.WriteString(fmt.Sprintf("%s: %s\n", key, spec))
		if err != nil {
			return err
		}
	}
	return nil
}

type minikube struct {
	configMaps []string
	services   []string
}

func startMiniKube() (*minikube, error) {
	r := shell.Timeout(30 * time.Second)
	var out bytes.Buffer
	err := r.AllOut(&out).Run("minikube", "status")
	output := string(out.Bytes())
	if !strings.Contains(output, "Running") {
		err = r.Run("minikube", "start")
		if err != nil {
			return nil, fmt.Errorf("unable to start minikube: %w", err)
		}
	}
	return &minikube{configMaps: make([]string, 0), services: make([]string, 0)}, nil
}

func (m *minikube) Close() error {
	r := shell.AllOut(os.Stdout).Timeout(2 * time.Second)
	for _, cm := range m.configMaps {
		err := r.Run("kubectl", "delete", "cm", cm)
		if err != nil {
			return fmt.Errorf("unable to delete configmap: %w", err)
		}
	}
	for _, svc := range m.services {
		err := r.Run("kubectl", "delete", "service", svc)
		if err != nil {
			return fmt.Errorf("unable to delete service: %w", err)
		}
		err = r.Run("kubectl", "delete", "deployment", svc)
		if err != nil {
			return fmt.Errorf("unable to delete deployment: %w", err)
		}
	}
	return nil
}

type ConfigMapTemplate struct {
	Name    string
	Entries map[string]string
}

func (m *minikube) addConfigMap(name string, entries map[string]string) error {
	tpl, err := template.ParseFiles("configmap.goyaml")
	if err != nil {
		return fmt.Errorf("unable parse configmap template: %w", err)
	}
	err = createAndApplyFile(tpl, ConfigMapTemplate{Name: name, Entries: entries})
	if err != nil {
		return fmt.Errorf("unable to create and apply configmap: %w", err)
	}
	m.configMaps = append(m.configMaps, name)
	return nil
}

type ServiceTemplate struct {
	Name string
}

func (m *minikube) addService(name string) error {
	tpl, err := template.ParseFiles("service.goyaml")
	if err != nil {
		return fmt.Errorf("unable to parse service template: %w", err)
	}
	err = createAndApplyFile(tpl, ServiceTemplate{Name: name})
	if err != nil {
		return fmt.Errorf("unable to create and apply service: %w", err)
	}
	m.services = append(m.services, name)
	return nil
}

func createAndApplyFile(tpl *template.Template, val interface{}) error {
	file, err := ioutil.TempFile("", "kube-resource")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	err = tpl.Execute(file, val)
	if err != nil {
		return fmt.Errorf("unable to execute template: %w", err)
	}
	return shell.Timeout(2*time.Second).Run("kubectl", "apply", "-f", file.Name())
}

type TmplConfig struct {
	EnvPrefix   string
	FilePath    string
	FilePrefix  string
	KubeEnabled bool
}

type testConfig struct {
	tmpl *template.Template
	dir  string
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
			"prefix": "{{ .FilePrefix }}",
			"json-ext": ".json",
			"url-ext": ".url"
        },
		{{- end}}
		{{- if .KubeEnabled }}
		"kubernetes": {
			"enabled": true
		},
		{{- end }}
		"thisisignored": 2
	}
}
`)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config template: %w", err)
	}
	return &testConfig{tpl, ""}, nil
}

func (t *testConfig) Close() error {
	if t.dir != "" {
		filepath.Walk(t.dir, func(path string, info os.FileInfo, err error) error {
			return os.Remove(path)
		})
		os.Remove(t.dir)
	}
	return nil
}

func (t *testConfig) configWith(conf TmplConfig) (string, error) {
	var tpl bytes.Buffer
	if err := t.tmpl.Execute(&tpl, conf); err != nil {
		return "", fmt.Errorf("unable to execute template: %w", err)
	}
	dir, err := ioutil.TempDir("", "config")
	if err != nil {
		t.Close()
		return "", fmt.Errorf("unable create tmp dir: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	file, err := os.Create(path)
	if err != nil {
		t.Close()
		return "", fmt.Errorf("unable to create file %s: %w", path, err)
	}
	_, err = tpl.WriteTo(file)
	if err != nil {
		t.Close()
		return "", fmt.Errorf("unable to write template to file %s: %w", path, err)
	}
	return path, nil
}

func nCheck(op string, t *testing.T, err error) {
	if err != nil {
		t.Errorf("op: '%s', unexpected error: %v", op, err)
	}
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}
