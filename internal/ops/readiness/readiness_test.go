package readiness_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ops/readiness"
)

func TestProbeOfflineReady(t *testing.T) {
	cfg := config.DefaultStorageConfig()
	rep := readiness.Probe(context.Background(), cfg, readiness.Options{})
	if !rep.OK || !rep.Ready || rep.Degraded {
		t.Fatalf("%#v", rep)
	}
	for _, b := range rep.Backends {
		if b.Role == "vector" && b.Status != readiness.StatusSkipped {
			t.Fatalf("vector should be skipped: %#v", b)
		}
		if b.Role != "vector" && b.Status != readiness.StatusReady {
			t.Fatalf("%#v", b)
		}
	}
}

func TestProbeFailInjectMetadata(t *testing.T) {
	t.Setenv("CONTEXT_FAIL_METADATA", "1")
	cfg := config.DefaultStorageConfig()
	rep := readiness.Probe(context.Background(), cfg, readiness.Options{})
	if rep.Ready || !rep.Degraded {
		t.Fatalf("%#v", rep)
	}
}
