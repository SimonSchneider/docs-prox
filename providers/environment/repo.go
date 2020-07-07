package environment

import (
	"os"
	"strings"

	"github.com/SimonSchneider/docs-prox/openapi"
)

// Config of the environment repository
type Config struct {
	Prefix string `json:"prefix"`
}

// Build the repository from the configuration
func (c *Config) Build() openapi.Repository {
	keys := make([]string, 0)
	specs := make(map[string]openapi.Spec)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if !strings.HasPrefix(pair[0], c.Prefix) {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(pair[0], c.Prefix), "_", "-"))
		keys = append(keys, key)
		specs[key] = openapi.NewRemoteSpec(pair[1])
	}
	return &environmentRepository{keys: keys, specs: specs}
}

type environmentRepository struct {
	keys  []string
	specs map[string]openapi.Spec
}

func (r *environmentRepository) Keys() ([]string, error) {
	return r.keys, nil
}

func (r *environmentRepository) Spec(key string) (openapi.Spec, error) {
	if val, ok := r.specs[key]; ok {
		return val, nil
	}
	return nil, openapi.KeyNotFoundError{Repo: "environmentRepository", Key: key}
}
