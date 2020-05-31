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
	delegates  []Repsitory
	cachedKeys map[string]Repsitory
	keys       []string
}

// AllOf returns a new Repository containing all the delegates
func AllOf(delegates ...Repsitory) Repsitory {
	return &multiRepository{delegates: delegates}
}

func (r *multiRepository) cacheKeys() {
	if r.keys != nil && r.cachedKeys != nil {
		return
	}
	fmt.Println("caching keys")
	r.cachedKeys = make(map[string]Repsitory)
	for _, delegate := range r.delegates {
		for _, key := range delegate.Keys() {
			if _, ok := r.cachedKeys[key]; !ok {
				r.cachedKeys[key] = delegate
				r.keys = append(r.keys, key)
			}
		}
	}
}

func (r *multiRepository) Keys() []string {
	r.cacheKeys()
	return r.keys
}

func (r *multiRepository) Spec(key string) (Spec, error) {
	r.cacheKeys()
	if val, ok := r.cachedKeys[key]; ok {
		return val.Spec(key)
	}
	return nil, fmt.Errorf("not found %s", key)
}
