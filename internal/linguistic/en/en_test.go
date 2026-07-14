package en_test

import (
	"testing"

	"github.com/fastygo/context/internal/linguistic/en"
	"github.com/fastygo/context/internal/linguistic/harness"
)

func TestContextLangENPassesHarness(t *testing.T) {
	t.Parallel()
	harness.RunContract(t, en.Ports())
}
