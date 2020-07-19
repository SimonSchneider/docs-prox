package openapi

import (
	"fmt"
	"math/rand"
	"testing"
)

type testSpec string

func (t testSpec) Get() ([]byte, error) {
	return []byte(t), nil
}

func rndSpec() Spec {
	return testSpec(fmt.Sprintf("test-spec-%d", rand.Int()))
}

func Test_cantOverwriteKeyOwnedByOtherSource(t *testing.T) {
	source1 := "existingSourceOwningKey"
	source2 := "source2"
	key1 := "key-owned-by1"
	spec1 := rndSpec()
	specAfterWrite := rndSpec()
	type args struct {
		source       string
		name         string
		specToPut    Spec
		specAfterPut Spec
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "can put into empty repo",
			args: args{
				source:       source1,
				name:         key1,
				specToPut:    spec1,
				specAfterPut: spec1,
			},
			wantErr: false,
		},
		{
			name: "can't overwrite spec owned by other source",
			args: args{
				source:       source2,
				name:         key1,
				specToPut:    rndSpec(),
				specAfterPut: spec1,
			},
			wantErr: true,
		},
		{
			name: "owning source can still replace spec",
			args: args{
				source:       source1,
				name:         key1,
				specToPut:    specAfterWrite,
				specAfterPut: specAfterWrite,
			},
			wantErr: false,
		},
	}
	r := NewCachedRepository()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := r.Put(tt.args.source, tt.args.name, tt.args.specToPut); (err != nil) != tt.wantErr {
				t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
			}
			if spec, err := r.Spec(tt.args.name); err != nil || spec != tt.args.specAfterPut {
				t.Errorf("Expected to find spec %v found %v and err %v", tt.args.specAfterPut, spec, err)
			}
		})
	}
}

func Test_cantDeleteKeyOwnedByOtherSource(t *testing.T) {
	r := NewCachedRepository()
	ownerSource := "owner"
	key := "key"
	spec := rndSpec()
	check("owner put", t, r.Put(ownerSource, key, spec))
	if err := r.Remove("not-owner", key); err == nil {
		t.Errorf("was able to remove key owned by another repo with no error")
	}
	if fs, err := r.Spec(key); fs != spec {
		t.Errorf("found spec %v differes from expected spec %v (error: %v)", fs, spec, err)
	}
	check("owner remove", t, r.Remove(ownerSource, key))
	if _, err := r.Spec(key); err == nil {
		t.Errorf("spec should not be found after removing sucessfully")
	}
}

func Test_cantReplaceKeyOwnedByOtherSource(t *testing.T) {
	r := NewCachedRepository()
	ownerSource := "owner"
	key := "key"
	spec := rndSpec()
	checkForSpec := func(expectedSpec Spec) {
		if fs, err := r.Spec(key); fs != expectedSpec {
			t.Errorf("found spec %v differes from expected spec %v (error: %v)", fs, expectedSpec, err)
		}
	}
	r.ReplaceAllOf(ownerSource, map[string]Spec{key: spec})
	checkForSpec(spec)
	r.ReplaceAllOf("not-owner", map[string]Spec{key: rndSpec()})
	checkForSpec(spec)
	newSpec := rndSpec()
	r.ReplaceAllOf(ownerSource, map[string]Spec{key: newSpec})
	checkForSpec(newSpec)
}

func Test_keysAreSorted(t *testing.T) {
	type sourceAndKey struct {
		source, key string
	}
	toPut := []sourceAndKey{
		{"s2", "B1"},
		{"s1", "b2"},
		{"s1", "a1"},
	}
	expected := []string{"a1", "B1", "b2"}
	r := NewCachedRepository()
	for _, p := range toPut {
		check("put", t, r.Put(p.source, p.key, rndSpec()))
	}
	keys := r.Keys()
	if len(keys) != len(expected) {
		t.Errorf("unexpected number of keys %d, expected %d", len(keys), len(expected))
	}
	for i := 0; i < len(expected); i++ {
		key := keys[i].Name
		expectedKey := expected[i]
		if key != expectedKey {
			t.Errorf("unexpected key %s in position %d, expected %s", key, i, expectedKey)
		}
	}
}

func check(operation string, t *testing.T, err error) {
	if err != nil {
		t.Errorf("op: '%s' expected no error got: %v", operation, err)
	}
}
