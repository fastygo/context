package refen_test

import (
	"testing"

	"github.com/fastygo/context/pkg/langtestkit"
	"github.com/fastygo/context/pkg/langtestkit/refen"
)

func TestContextLangENPassesLangtestkit(t *testing.T) {
	t.Parallel()
	langtestkit.RunContract(t, refen.Ports())
}
