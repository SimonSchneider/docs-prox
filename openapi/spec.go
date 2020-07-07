package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Spec is the openApi spec abstraction
type Spec interface {
	JSONSpec() (interface{}, error)
}

type staticSpec struct {
	spec interface{}
}

// NewStaticSpec creates a static spec with an inmemory spec
func NewStaticSpec(spec interface{}) Spec {
	return &staticSpec{spec: spec}
}

func (s *staticSpec) JSONSpec() (interface{}, error) {
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

func (s *remoteSpec) JSONSpec() (interface{}, error) {
	resp, err := s.client.Get(s.url)
	if err != nil {
		return nil, fmt.Errorf("remoteSpec: unable to fetch spec from %s: %w", s.url, err)
	}
	var result interface{}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}
