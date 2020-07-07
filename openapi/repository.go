package openapi

import (
	"fmt"
	"sort"
)

// Repository abstracts a documentation provider holding keys and specs
type Repository interface {
	Keys() ([]string, error)
	Spec(key string) (Spec, error)
}

type KeyNotFoundError struct {
	Repo string
	Key  string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("%s: Key %s not found", e.Repo, e.Key)
}

type multiRepository struct {
	delegates []Repository
}

// AllOf returns a new Repository containing all the delegates
func AllOf(delegates ...Repository) Repository {
	return &multiRepository{delegates: delegates}
}

func (r *multiRepository) Keys() ([]string, error) {
	keySet := make(map[string]interface{})
	keys := make([]string, 0)
	for _, delegate := range r.delegates {
		dKeys, err := delegate.Keys()
		if err != nil {
			fmt.Printf("error collecting keys, ignoring repository: %v\n", err)
			continue
		}
		for _, key := range dKeys {
			if _, ok := keySet[key]; !ok {
				keySet[key] = nil
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (r *multiRepository) Spec(key string) (Spec, error) {
	for _, delegate := range r.delegates {
		keys, err := delegate.Keys()
		if err != nil {
			fmt.Printf("error collecting keys, ignoring repository: %v\n", err)
			continue
		}
		for _, k := range keys {
			if k == key {
				return delegate.Spec(k)
			}
		}
	}
	return nil, KeyNotFoundError{Repo: "multiRepository", Key: key}
}
