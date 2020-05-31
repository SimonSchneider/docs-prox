package openapi

import (
	"fmt"
)

// Repsitory abstracts a documentation provider holding keys and specs
type Repsitory interface {
	Keys() []string
	Spec(key string) (Spec, error)
}

type staticRepository struct {
	specs map[string]Spec
	keys  []string
}

// NewStaticRepo creates a repo with static map of specs
func NewStaticRepo(specs map[string]Spec) Repsitory {
	keys := make([]string, 0, len(specs))
	for key := range specs {
		keys = append(keys, key)
	}
	return &staticRepository{specs: specs, keys: keys}
}

func (r *staticRepository) Keys() []string {
	return r.keys
}

func (r *staticRepository) Spec(key string) (Spec, error) {
	if val, ok := r.specs[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("not found %s", key)
}

type multiRepository struct {
	delegates []Repsitory
}

// AllOf returns a new Repository containing all the delegates
func AllOf(delegates ...Repsitory) Repsitory {
	return &multiRepository{delegates: delegates}
}

func (r *multiRepository) Keys() []string {
	keySet := make(map[string]interface{})
	keys := make([]string, 0)
	for _, delegate := range r.delegates {
		for _, key := range delegate.Keys() {
			if _, ok := keySet[key]; !ok {
				keySet[key] = nil
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func (r *multiRepository) Spec(key string) (Spec, error) {
	for _, delegate := range r.delegates {
		for _, k := range delegate.Keys() {
			if k == key {
				return delegate.Spec(key)
			}
		}
	}
	return nil, fmt.Errorf("not found %s", key)
}
