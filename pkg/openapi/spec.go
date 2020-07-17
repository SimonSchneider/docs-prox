package openapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// Spec is the openApi spec abstraction
type Spec interface {
	Get() ([]byte, error)
}

type cachedSpec struct {
	delegate  Spec
	ttl       time.Duration
	mu        sync.RWMutex
	expiresAt time.Time
	resp      []byte
}

func (c *cachedSpec) Get() ([]byte, error) {
	c.mu.RLock()
	if r, ok := c.tryGetFromCache(); ok {
		c.mu.RUnlock()
		return r, nil
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.tryGetFromCache(); ok {
		return r, nil
	}
	return c.getFromDelegateAndUpdateCache()
}

func (c *cachedSpec) tryGetFromCache() ([]byte, bool) {
	if c.resp != nil && c.expiresAt.After(time.Now()) {
		return c.resp, true
	}
	return nil, false
}

func (c *cachedSpec) getFromDelegateAndUpdateCache() ([]byte, error) {
	resp, err := c.delegate.Get()
	if err != nil {
		c.resp = nil
		return nil, err
	}
	c.resp = resp
	c.expiresAt = time.Now().Add(c.ttl)
	return resp, nil
}

// Cached returns a spec that wraps and caches the delegate spec
func Cached(delegate Spec, ttl time.Duration) Spec {
	return &cachedSpec{
		delegate: delegate,
		ttl:      ttl,
		mu:       sync.RWMutex{},
		resp:     nil,
	}
}

type remoteSpec struct {
	client *http.Client
	url    string
}

// NewRemoteSpec creates a spec that is proxied from a remote url
func NewRemoteSpec(url string) Spec {
	return &remoteSpec{client: &http.Client{}, url: url}
}

func (s *remoteSpec) Get() ([]byte, error) {
	resp, err := s.client.Get(s.url)
	if err != nil {
		return nil, fmt.Errorf("remoteSpec: unable to fetch spec from %s: %w", s.url, err)
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

//NewCachedRemoteSpec is a convenience constructor for a cached remote spec
func NewCachedRemoteSpec(url string, ttl time.Duration) Spec {
	return Cached(NewRemoteSpec(url), ttl)
}
