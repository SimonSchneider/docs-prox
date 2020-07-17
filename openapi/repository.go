package openapi

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Repository abstracts a documentation provider holding keys and specs
type Repository interface {
	Keys() []SpecMetadata
	Spec(key string) (Spec, error)
}

type SpecMetadata struct {
	Key, Name string
}

func SpecMetadataOf(name string) SpecMetadata {
	return SpecMetadata{
		Key:  strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Name: name,
	}
}

// KeyNotFoundError is returned when a key is not found
type KeyNotFoundError struct {
	Repo string
	Key  string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("%s: SpecMetadata %s not found", e.Repo, e.Key)
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
	specs   *SortedMap
}

type KeySpec struct {
	SpecMetadata
	Spec
}

// NewCachedRepository creats a new CachedRepo
func NewCachedRepository() SpecRepoStore {
	return &cachedRepository{
		mu:      &sync.RWMutex{},
		sources: make(map[string]map[string]struct{}),
		specs:   NewSortedMap(),
	}
}

func (r *cachedRepository) Keys() []SpecMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]SpecMetadata, 0, r.specs.Len())
	r.specs.RangeOver(func(k string, v KeySpec) {
		keys = append(keys, v.SpecMetadata)
	})
	return keys
}

func (r *cachedRepository) Spec(key string) (Spec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if spec, ok := r.specs.Get(key); ok {
		return spec, nil
	}
	return nil, KeyNotFoundError{Repo: "cachedRepo", Key: key}
}

func (r *cachedRepository) Put(source, name string, spec Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sources[source]; !ok {
		r.sources[source] = make(map[string]struct{})
	}
	key := SpecMetadataOf(name)
	r.sources[source][key.Key] = struct{}{}
	r.specs.Set(key.Key, KeySpec{
		SpecMetadata: key,
		Spec:         spec,
	})
}

func (r *cachedRepository) ReplaceAllOf(source string, specs map[string]Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	multi := r.specs.NewMultiChange()
	defer multi.Finished()
	for key := range r.sources[source] {
		multi.Delete(key)
	}
	r.sources[source] = make(map[string]struct{}, len(specs))
	for name, spec := range specs {
		key := SpecMetadataOf(name)
		r.sources[source][key.Key] = struct{}{}
		multi.Set(key.Key, KeySpec{
			SpecMetadata: key,
			Spec:         spec,
		})
	}
}

func (r *cachedRepository) Remove(source, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := SpecMetadataOf(name)
	delete(r.sources[source], key.Key)
	r.specs.Delete(key.Key)
}

func (r *cachedRepository) RemoveAllOf(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	multi := r.specs.NewMultiChange()
	defer multi.Finished()
	for key := range r.sources[source] {
		multi.Delete(key)
	}
	delete(r.sources, source)
}

type SortedMap struct {
	m map[string]KeySpec
	l []string
}

func NewSortedMap() *SortedMap {
	return &SortedMap{
		m: make(map[string]KeySpec),
		l: make([]string, 0),
	}
}

func (s *SortedMap) RangeOver(fun func(string, KeySpec)) {
	for _, k := range s.l {
		fun(k, s.m[k])
	}
}

func (s *SortedMap) Get(key string) (KeySpec, bool) {
	val, ok := s.m[key]
	return val, ok
}

func (s *SortedMap) Len() int {
	return len(s.m)
}

func (s *SortedMap) setUnsafe(key string, val KeySpec) {
	s.m[key] = val
}

func (s *SortedMap) deleteUnsafe(key string) {
	delete(s.m, key)
}

func (s *SortedMap) Set(key string, val KeySpec) {
	s.setUnsafe(key, val)
	s.updateSort()
}

func (s *SortedMap) Delete(key string) {
	s.deleteUnsafe(key)
	s.updateSort()
}

func (s *SortedMap) updateSort() {
	s.l = s.l[0:0]
	for k := range s.m {
		s.l = append(s.l, k)
	}
	sort.Strings(s.l)
}

type MultiChange struct {
	s *SortedMap
}

func (m *MultiChange) Set(key string, val KeySpec) {
	m.s.setUnsafe(key, val)
}

func (m *MultiChange) Delete(key string) {
	m.s.deleteUnsafe(key)
}

func (m *MultiChange) Finished() {
	m.s.updateSort()
}

func (s *SortedMap) NewMultiChange() *MultiChange {
	return &MultiChange{
		s: s,
	}
}
