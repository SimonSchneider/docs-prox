package openapi

import (
	"fmt"
	"sort"
	"sync"
)

// Repository abstracts a documentation provider holding keys and specs
type Repository interface {
	Keys() []string
	Spec(key string) (Spec, error)
}

type KeyNotFoundError struct {
	Repo string
	Key  string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("%s: Key %s not found", e.Repo, e.Key)
}

type sortedRepo struct {
	delegate Repository
}

func Sorted(delegate Repository) Repository {
	return &sortedRepo{delegate: delegate}
}

func (s *sortedRepo) Keys() []string {
	keys := s.delegate.Keys()
	sort.Strings(keys)
	return keys
}

func (s *sortedRepo) Spec(key string) (Spec, error) {
	return s.delegate.Spec(key)
}

type ApiStore interface {
	Put(source, key string, spec Spec)
	ReplaceAllOf(source string, specs map[string]Spec)
	Remove(source, key string)
	RemoveAllOf(source string)
}

type CachedRepository struct {
	mu      *sync.RWMutex
	sources map[string]map[string]struct{}
	specs   map[string]Spec
}

func NewCachedRepository() *CachedRepository {
	return &CachedRepository{
		mu:      &sync.RWMutex{},
		sources: make(map[string]map[string]struct{}),
		specs:   make(map[string]Spec),
	}
}

func (r *CachedRepository) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.specs))
	for key := range r.specs {
		keys = append(keys, key)
	}
	return keys
}

func (r *CachedRepository) Spec(key string) (Spec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if spec, ok := r.specs[key]; ok {
		return spec, nil
	}
	return nil, KeyNotFoundError{Repo: "cachedRepo", Key: key}
}

func (r *CachedRepository) Put(source, key string, spec Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sources[source]; !ok {
		r.sources[source] = make(map[string]struct{})
	}
	r.sources[source][key] = struct{}{}
	r.specs[key] = spec
}

func (r *CachedRepository) ReplaceAllOf(source string, specs map[string]Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.sources[source] {
		delete(r.specs, key)
	}
	r.sources[source] = make(map[string]struct{}, len(specs))
	for key, spec := range specs {
		r.sources[source][key] = struct{}{}
		r.specs[key] = spec
	}
}

func (r *CachedRepository) Remove(source, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sources[source], key)
	delete(r.specs, key)
}

func (r *CachedRepository) RemoveAllOf(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.sources[source] {
		delete(r.specs, key)
	}
	delete(r.sources, source)
}
