package static

import (
	"github.com/SimonSchneider/docs-prox/openapi"
)

type staticRepository struct {
	specs map[string]openapi.Spec
	keys  []string
}

// NewStaticRepo creates a repo with static map of specs
func NewStaticRepo(specs map[string]openapi.Spec) openapi.Repository {
	keys := make([]string, 0, len(specs))
	for key := range specs {
		keys = append(keys, key)
	}
	return &staticRepository{specs: specs, keys: keys}
}

func (r *staticRepository) Keys() ([]string, error) {
	return r.keys, nil
}

func (r *staticRepository) Spec(key string) (openapi.Spec, error) {
	if val, ok := r.specs[key]; ok {
		return val, nil
	}
	return nil, openapi.KeyNotFoundError{Repo: "staticRepository", Key: key}
}
