package environment

import (
	"fmt"
	"os"
	"strings"

	o "github.com/SimonSchneider/docs-prox/openapi"
)

type environmentRepository struct {
	keys  []string
	specs map[string]o.Spec
}

func NewEnvironmentRepsitory(prefix string) o.Repsitory {
	// access the directory to validate access
	keys := make([]string, 0)
	specs := make(map[string]o.Spec)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if !strings.HasPrefix(pair[0], prefix) {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(pair[0], prefix), "_", "-"))
		keys = append(keys, key)
		specs[key] = o.NewRemoteSpec(pair[1])
	}
	return &environmentRepository{keys: keys, specs: specs}
}

func (r *environmentRepository) Keys() []string {
	return r.keys
}

func (r *environmentRepository) Spec(key string) (o.Spec, error) {
	if val, ok := r.specs[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("not found")
}
