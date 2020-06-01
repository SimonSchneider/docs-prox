package openapi

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Repository abstracts a documentation provider holding keys and specs
type Repository interface {
	Keys() []string
	Spec(key string) (Spec, error)
}

type staticRepository struct {
	specs map[string]Spec
	keys  []string
}

// NewStaticRepo creates a repo with static map of specs
func NewStaticRepo(specs map[string]Spec) Repository {
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
	delegates []Repository
}

// AllOf returns a new Repository containing all the delegates
func AllOf(delegates ...Repository) Repository {
	return &multiRepository{delegates: delegates}
}

func (r *multiRepository) Keys() []string {
	keys := make(chan string)
	var wg sync.WaitGroup
	wg.Add(len(r.delegates))
	for _, delegate := range r.delegates {
		go func(repo Repository) {
			defer wg.Done()
			for _, k := range repo.Keys() {
				keys <- k
			}
		}(delegate)
	}
	go func() {
		wg.Wait()
		close(keys)
	}()
	return collectKeys(keys)
}

func collectKeys(incoming <-chan string) []string {
	keySet := make(map[string]interface{})
	keys := make([]string, 0)
	for key := range incoming {
		if _, ok := keySet[key]; !ok {
			keySet[key] = nil
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func (r *multiRepository) Spec(key string) (Spec, error) {
	resChan := make(chan result)
	var wg sync.WaitGroup
	wg.Add(len(r.delegates))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, delegate := range r.delegates {
		go func(repo Repository) {
			defer wg.Done()
			getSpecAsync(ctx, repo, key, resChan)
		}(delegate)
	}
	go func() {
		wg.Wait()
		close(resChan)
	}()
	if res, ok := <-resChan; ok {
		return res.spec, res.err
	}
	return nil, fmt.Errorf("not found")
}

func getSpecAsync(ctx context.Context, repo Repository, key string, res chan<- result) {
	for _, k := range repo.Keys() {
		select {
		case <-ctx.Done():
			return
		default:
			if k == key {
				spec, err := repo.Spec(key)
				res <- result{spec: spec, err: err}
				return
			}
		}
	}
}

type result struct {
	spec Spec
	err  error
}
