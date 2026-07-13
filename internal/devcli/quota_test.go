package devcli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/policy"
)

func TestQuotaStatusAndDenyPack(t *testing.T) {
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# Alpha\n\nZEBRA42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_q", "Quota"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_q", ""); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONTEXT_QUOTA_MAX_PACKS", "1")
	t.Setenv("CONTEXT_QUOTA_MAX_RUNS", "")
	t.Setenv("CONTEXT_QUOTA_MAX_CHUNKS", "")
	t.Setenv("CONTEXT_QUOTA_SOFT_ASK_PERCENT", "80")

	st, err := devcli.QuotaStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if st.Decision != policy.DecisionAllow || !st.Limits.Enabled() {
		t.Fatalf("status: %#v", st)
	}

	if _, err := devcli.BuildPack(data, "proj_q", "ZEBRA42", ""); err != nil {
		t.Fatal(err)
	}
	// At hard limit: second pack denied.
	_, err = devcli.BuildPack(data, "proj_q", "ZEBRA42", "")
	if err == nil || !apperr.Is(err, apperr.Permission) {
		t.Fatalf("want permission deny, got %v", err)
	}
	if !strings.Contains(err.Error(), "quota deny") {
		t.Fatalf("msg: %v", err)
	}

	st, err = devcli.QuotaStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if st.Decision != policy.DecisionDeny || st.OK {
		t.Fatalf("want deny status: %#v", st)
	}

	m, err := devcli.Metrics(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Quota == nil || m.Quota.Decision != policy.DecisionDeny {
		t.Fatalf("metrics quota: %#v", m.Quota)
	}
}
