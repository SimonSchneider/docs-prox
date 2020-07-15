package environment

import (
	"os"
	"strings"
	"time"

	"github.com/SimonSchneider/docs-prox/openapi"
)

// Configure the repository from the configuration
func Configure(store openapi.SpecStore, prefix string) {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], prefix) {
			key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(pair[0], prefix), "_", "-"))
			spec := openapi.NewCachedRemoteSpec(pair[1], 20*time.Second)
			store.Put("env", key, spec)
		}
	}
}
