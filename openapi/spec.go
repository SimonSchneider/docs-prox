package openapi

import (
	"encoding/json"
	"net/http"
)

type Spec interface {
	JsonSpec() (interface{}, error)
}

type staticSpec struct {
	spec interface{}
}

func NewStaticSpec(spec interface{}) *staticSpec {
	return &staticSpec{spec: spec}
}

func (s *staticSpec) JsonSpec() (interface{}, error) {
	return s.spec, nil
}

type remoteSpec struct {
	url string
}

func NewRemoteSpec(url string) *remoteSpec {
	return &remoteSpec{url: url}
}

func (s *remoteSpec) JsonSpec() (interface{}, error) {
	resp, err := http.Get(s.url)
	if err != nil {
		return nil, err
	}
	var result interface{}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}
