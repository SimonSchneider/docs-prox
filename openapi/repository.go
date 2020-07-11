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

// KeyNotFoundError is returned when a key is not found
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

// Sorted returns a repo that sorts the keys in the underlying repo
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

// SpecStore is a concurrent Spec store
type SpecStore interface {
	Put(source, key string, spec Spec)
	ReplaceAllOf(source string, specs map[string]Spec)
	Remove(source, key string)
	RemoveAllOf(source string)
}

type loggingSpecStore struct {
	delegate SpecStore
}

// Logging wraps the Spec Store in a logging spec store
func Logging(delegate SpecStore) SpecStore {
	return &loggingSpecStore{delegate: delegate}
}

func (l *loggingSpecStore) Put(source, key string, spec Spec) {
	fmt.Printf("Putting (%s - %s)\n", source, key)
	l.delegate.Put(source, key, spec)
}

func (l *loggingSpecStore) ReplaceAllOf(source string, specs map[string]Spec) {
	fmt.Printf("Replacing all (%s)\n", source)
	l.delegate.ReplaceAllOf(source, specs)
}

func (l *loggingSpecStore) Remove(source, key string) {
	fmt.Printf("Removing (%s - %s)\n", source, key)
	l.delegate.Remove(source, key)
}

func (l *loggingSpecStore) RemoveAllOf(source string) {
	fmt.Printf("Removing all (%s)\n", source)
	l.delegate.RemoveAllOf(source)
}

//SpecRepoStore combined
type SpecRepoStore interface {
	SpecStore
	Repository
}

// cachedRepository is the root implementation for Repository and SpecStore
type cachedRepository struct {
	mu      *sync.RWMutex
	sources map[string]map[string]struct{}
	specs   map[string]Spec
}

// NewCachedRepository creats a new CachedRepo
func NewCachedRepository() SpecRepoStore {
	return &cachedRepository{
		mu:      &sync.RWMutex{},
		sources: make(map[string]map[string]struct{}),
		specs:   make(map[string]Spec),
	}
}

func (r *cachedRepository) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.specs))
	for key := range r.specs {
		keys = append(keys, key)
	}
	return keys
}

func (r *cachedRepository) Spec(key string) (Spec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if spec, ok := r.specs[key]; ok {
		return spec, nil
	}
	return nil, KeyNotFoundError{Repo: "cachedRepo", Key: key}
}

func (r *cachedRepository) Put(source, key string, spec Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sources[source]; !ok {
		r.sources[source] = make(map[string]struct{})
	}
	r.sources[source][key] = struct{}{}
	r.specs[key] = spec
}

func (r *cachedRepository) ReplaceAllOf(source string, specs map[string]Spec) {
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

func (r *cachedRepository) Remove(source, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sources[source], key)
	delete(r.specs, key)
}

func (r *cachedRepository) RemoveAllOf(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.sources[source] {
		delete(r.specs, key)
	}
	delete(r.sources, source)
}
