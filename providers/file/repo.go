package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/SimonSchneider/docs-prox/openapi"
)

// Config of file repo
type Config struct {
	Path   string `json:"path"`
	Prefix string `json:"prefix"`
}

// Build a repository from the config
func (c *Config) Build() (openapi.Repsitory, error) {
	if _, err := os.Stat(c.Path); err != nil {
		return nil, err
	}
	return &fileRepsitory{path: c.Path, prefix: c.Prefix}, nil
}

type fileRepsitory struct {
	path   string
	prefix string
}

func (r *fileRepsitory) Keys() []string {
	var keys []string
	err := filepath.Walk(r.path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasPrefix(info.Name(), r.prefix) && strings.HasSuffix(info.Name(), ".json") {
			key := strings.TrimSuffix(strings.TrimPrefix(info.Name(), r.prefix), ".json")
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil {
		return []string{}
	}
	return keys
}

func (r *fileRepsitory) Spec(key string) (openapi.Spec, error) {
	fileName := r.prefix + key + ".json"
	filePath := filepath.Join(r.path, fileName)
	return &fileSpec{filePath}, nil
}

type fileSpec struct {
	path string
}

func (s *fileSpec) JSONSpec() (interface{}, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	var result interface{}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&result)
	return result, err
}