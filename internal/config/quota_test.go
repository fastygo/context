package config_test

import (
	"testing"

	"github.com/fastygo/context/internal/config"
)

func TestLoadQuotaLimitsFromEnv(t *testing.T) {
	t.Setenv("CONTEXT_QUOTA_MAX_CHUNKS", "100")
	t.Setenv("CONTEXT_QUOTA_MAX_PACKS", "20")
	t.Setenv("CONTEXT_QUOTA_MAX_RUNS", "10")
	t.Setenv("CONTEXT_QUOTA_SOFT_ASK_PERCENT", "75")
	lim, err := config.LoadQuotaLimitsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if lim.MaxChunks != 100 || lim.MaxPacks != 20 || lim.MaxRuns != 10 || lim.SoftAskPercent != 75 {
		t.Fatalf("%#v", lim)
	}
}

func TestLoadQuotaLimitsUnset(t *testing.T) {
	t.Setenv("CONTEXT_QUOTA_MAX_CHUNKS", "")
	t.Setenv("CONTEXT_QUOTA_MAX_PACKS", "")
	t.Setenv("CONTEXT_QUOTA_MAX_RUNS", "")
	t.Setenv("CONTEXT_QUOTA_SOFT_ASK_PERCENT", "")
	lim, err := config.LoadQuotaLimitsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if lim.Enabled() {
		t.Fatalf("want disabled: %#v", lim)
	}
}
