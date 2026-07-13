package contextkit_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPackageImportsExcludeInternal(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}\n{{join .TestImports \"\\n\"}}", "github.com/fastygo/context/pkg/contextkit").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "github.com/fastygo/context/internal/") {
			t.Fatalf("pkg/contextkit must not import internal: %s", line)
		}
	}
}
