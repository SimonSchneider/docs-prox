package environment

import (
	"fmt"
	"os"
	"strings"

	"github.com/SimonSchneider/docs-prox/openapi"
)

type environmentRepository struct {
	keys  []string
	specs map[string]openapi.Spec
}

// NewEnvironmentRepsitory returns a new repo fetching config form env variables with a given prefix
func NewEnvironmentRepsitory(prefix string) openapi.Repsitory {
	keys := make([]string, 0)
	specs := make(map[string]openapi.Spec)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if !strings.HasPrefix(pair[0], prefix) {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(pair[0], prefix), "_", "-"))
		keys = append(keys, key)
		specs[key] = openapi.NewRemoteSpec(pair[1])
	}
	return &environmentRepository{keys: keys, specs: specs}
}

func (r *environmentRepository) Keys() []string {
	return r.keys
}

func (r *environmentRepository) Spec(key string) (openapi.Spec, error) {
	if val, ok := r.specs[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("not found")
}
