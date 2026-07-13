package quota_test

import (
	"testing"

	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/policy/quota"
)

func TestEvaluateAllowAskDeny(t *testing.T) {
	limits := quota.Limits{MaxChunks: 10, MaxPacks: 5, MaxRuns: 3, SoftAskPercent: 80}

	st := quota.Evaluate(limits, quota.Usage{Chunks: 1, Packs: 1, Runs: 1})
	if st.Decision != policy.DecisionAllow || !st.OK {
		t.Fatalf("%#v", st)
	}

	st = quota.Evaluate(limits, quota.Usage{Chunks: 8, Packs: 1, Runs: 1})
	if st.Decision != policy.DecisionAsk || len(st.Breaches) < 1 {
		t.Fatalf("want ask: %#v", st)
	}

	st = quota.Evaluate(limits, quota.Usage{Chunks: 10, Packs: 1, Runs: 1})
	if st.Decision != policy.DecisionDeny || st.OK {
		t.Fatalf("want deny: %#v", st)
	}
}

func TestUnlimitedWhenZero(t *testing.T) {
	st := quota.Evaluate(quota.Limits{}, quota.Usage{Chunks: 999})
	if st.Decision != policy.DecisionAllow || !st.OK {
		t.Fatalf("%#v", st)
	}
}

func TestBlocksResource(t *testing.T) {
	limits := quota.Limits{MaxRuns: 2}
	if !quota.BlocksResource(limits, quota.Usage{Runs: 2}, "runs") {
		t.Fatal("already at limit")
	}
	if quota.BlocksResource(limits, quota.Usage{Runs: 1}, "runs") {
		t.Fatal("under limit")
	}
}
