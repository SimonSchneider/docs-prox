package openapi

import (
	"fmt"
	"testing"

	"github.com/SimonSchneider/docs-prox/providers/static"
)

type brokenRepoT struct {
}

func (b brokenRepoT) Keys() ([]string, error) {
	return []string{brokenKey}, nil
}

func (b brokenRepoT) Spec(key string) (Spec, error) {
	return nil, fmt.Errorf("this is broken")
}

var (
	brokenKey      = "brokenKey"
	notFoundKey    = "notFoundKey"
	foundKey       = "foundKey"
	foundSpec      = NewRemoteSpec("url1")
	duplicatedSpec = NewRemoteSpec("url2")
	emptyRepo      = static.NewStaticRepo(map[string]Spec{})
	brokenRepo     = brokenRepoT{}
	repoWithKey    = static.NewStaticRepo(map[string]Spec{foundKey: foundSpec})
	duplicatedRepo = static.NewStaticRepo(map[string]Spec{foundKey: duplicatedSpec})
	fullRepo       = AllOf(emptyRepo, brokenRepo, repoWithKey, duplicatedRepo)
	// fullRepo = AsyncAllOf(emptyRepo, brokenRepo, repoWithKey, duplicatedRepo)
)

func TestKeys(t *testing.T) {
	keys, _ := fullRepo.Keys()
	if len(keys) != 2 {
		t.Errorf("too many keys %s", keys)
	}
	for _, k := range keys {
		if k != foundKey && k != brokenKey {
			t.Errorf("incorrect Key %s found", k)
		}
	}
}

func TestBrokenSpec(t *testing.T) {
	spec, err := fullRepo.Spec(brokenKey)
	if err == nil {
		t.Errorf("expected error here got %s, %s", spec, err)
	}
}

func TestOkSpec(t *testing.T) {
	spec, err := fullRepo.Spec(foundKey)
	if err != nil {
		t.Errorf("expected error here got %s, %s", spec, err)
	}
	if spec != foundSpec && spec != duplicatedSpec {
		t.Errorf("unexpected spec %s", spec)
	}
}

func BenchmarkKeys(b *testing.B) {
	for n := 0; n < b.N; n++ {
		fullRepo.Keys()
	}
}

func BenchmarkSpec(b *testing.B) {
	for n := 0; n < b.N; n++ {
		fullRepo.Spec(foundKey)
	}
}
