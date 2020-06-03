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
	sort.Strings(keys)
	return keys
}

func (r *multiRepository) Spec(key string) (Spec, error) {
	for _, delegate := range r.delegates {
		for _, k := range delegate.Keys() {
			if k == key {
				return delegate.Spec(k)
			}
		}
	}
	return nil, fmt.Errorf("not found")
}

type asyncMultiRepository struct {
	delegates []Repository
}

// AsyncAllOf returns a new Repository containing all the delegates
// and an async implementation of keys() and spec(key string)
func AsyncAllOf(delegates ...Repository) Repository {
	return &asyncMultiRepository{delegates: delegates}
}

func (r *asyncMultiRepository) Keys() []string {
	keys := make(chan string)
	ctx := context.Background()
	r.keysAsync(ctx, keys)
	return collectKeys(keys)
}

func (r *asyncMultiRepository) keysAsync(ctx context.Context, keys chan<- string) {
	var wg sync.WaitGroup
	wg.Add(len(r.delegates))
	for _, delegate := range r.delegates {
		go func(repo Repository) {
			defer wg.Done()
			for _, k := range repo.Keys() {
				select {
				case <-ctx.Done():
					return
				case keys <- k:
				}
			}
		}(delegate)
	}
	go func() {
		wg.Wait()
		close(keys)
	}()
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

func (r *asyncMultiRepository) Spec(key string) (Spec, error) {
	spec := make(chan Spec)
	err := make(chan error)
	defer close(spec)
	defer close(err)
	ctx := context.Background()
	r.specAsync(ctx, spec, err, key)
	select {
	case s := <-spec:
		return s, nil
	case e := <-err:
		return nil, e
	}
}

func (r *asyncMultiRepository) specAsync(ctx context.Context, outCh chan<- Spec, errCh chan<- error, key string) {
	resultCh := make(chan result)
	var wg sync.WaitGroup
	cctx, cancel := context.WithCancel(ctx)
	wg.Add(len(r.delegates))
	for _, delegate := range r.delegates {
		go func(repo Repository) {
			defer wg.Done()
			for _, k := range repo.Keys() {
				select {
				case <-cctx.Done():
					return
				default:
					if k == key {
						spec, err := repo.Spec(k)
						resultCh <- result{spec: spec, err: err}
					}
				}
			}
		}(delegate)
	}
	go func() {
		defer close(resultCh)
		wg.Wait()
	}()
	go func() {
		defer cancel()
		if res, ok := <-resultCh; ok {
			if res.spec != nil {
				outCh <- res.spec
			} else {
				errCh <- res.err
			}
		} else {
			errCh <- fmt.Errorf("not found")
		}
	}()
}

type result struct {
	spec Spec
	err  error
}
