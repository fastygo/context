// Package failinject provides offline failure injection for readiness and
// adapter open paths (Chunk 29 / roadmap Failure Injection subset).
package failinject

import (
	"os"
	"strings"

	"github.com/fastygo/context/internal/apperr"
)

// Component names match CONTEXT_FAIL_<COMPONENT> env keys.
const (
	Metadata  = "metadata"
	Vector    = "vector"
	Sparse    = "sparse"
	Embedder  = "embedder"
	Artifact  = "artifact"
	Completer = "completer"
)

// Enabled reports whether CONTEXT_FAIL_<COMPONENT>=1|true is set.
func Enabled(component string) bool {
	key := "CONTEXT_FAIL_" + strings.ToUpper(component)
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true")
}

// Check returns Unavailable when failure injection is enabled for component.
func Check(component string) error {
	if !Enabled(component) {
		return nil
	}
	return apperr.New(apperr.Unavailable, component+" failure injected")
}
