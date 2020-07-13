package openapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Spec is the openApi spec abstraction
type Spec interface {
	Get() ([]byte, error)
}

type staticSpec struct {
	spec []byte
}

// NewStaticSpec creates a static spec with an inmemory spec
func NewStaticSpec(spec []byte) Spec {
	return &staticSpec{spec: spec}
}

func (s *staticSpec) Get() ([]byte, error) {
	return s.spec, nil
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
