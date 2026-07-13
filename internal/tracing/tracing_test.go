package tracing_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/tracing"
)

func TestEventRejectsZeroTimestamp(t *testing.T) {
	t.Parallel()
	e := tracing.Event{
		ID:        "e1",
		ProjectID: "p1",
		Type:      tracing.EventRunStarted,
	}
	if err := e.Validate(); err == nil {
		t.Fatal("expected zero timestamp to fail")
	}
	e.Timestamp = time.Unix(1, 0).UTC()
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
}
