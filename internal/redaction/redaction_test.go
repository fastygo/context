package redaction_test

import (
	"strings"
	"testing"

	"github.com/fastygo/context/internal/redaction"
)

func TestDefaultRedactsSecrets(t *testing.T) {
	in := strings.Join([]string{
		"Authorization: Bearer sk-live-ABCDEFGH123456",
		`api_key=supersecretvalue99`,
		`password: hunter2xx`,
		"contact me@example.com please",
	}, "\n")
	out, rep := (redaction.Default{}).Redact(in)
	if !rep.Applied || rep.Count < 4 {
		t.Fatalf("report: %#v out=%q", rep, out)
	}
	for _, bad := range []string{"sk-live-ABCDEFGH123456", "supersecretvalue99", "hunter2xx", "me@example.com"} {
		if strings.Contains(out, bad) {
			t.Fatalf("leak %q in %q", bad, out)
		}
	}
	if !strings.Contains(out, redaction.Replacement) {
		t.Fatal("missing replacement")
	}
}

func TestApplyRespectsEnv(t *testing.T) {
	t.Setenv("CONTEXT_REDACT", "0")
	out, rep := redaction.Apply("password=hunter2xx")
	if rep.Applied || out != "password=hunter2xx" {
		t.Fatalf("%q %#v", out, rep)
	}
	t.Setenv("CONTEXT_REDACT", "1")
	out, rep = redaction.Apply("password=hunter2xx")
	if !rep.Applied || strings.Contains(out, "hunter2xx") {
		t.Fatalf("%q %#v", out, rep)
	}
}
