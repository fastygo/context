// Package localecho is a deterministic offline Completer that cites pack
// evidence (Chunk 27). It is not a live LLM and does not require network.
package localecho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/models"
)

const (
	// Kind is the CONTEXT_COMPLETER_KIND value.
	Kind = "localecho"
	// ProviderID is recorded on ModelCall.
	ProviderID = "localecho"
	// ModelVersion pins the echo citation format.
	ModelVersion = "localecho-cite-v1"
)

// Completer answers by citing selected evidence surfaces and pack checksum.
type Completer struct{}

func (Completer) Complete(ctx context.Context, req models.CompletionRequest) (models.CompletionResult, error) {
	if err := ctx.Err(); err != nil {
		return models.CompletionResult{}, err
	}
	var b strings.Builder
	b.WriteString("localecho:")
	b.WriteString(string(req.Pack.ID))
	b.WriteString(" checksum=")
	b.WriteString(string(req.Pack.Checksum))
	b.WriteByte('\n')
	if len(req.Pack.EvidenceItems) == 0 {
		b.WriteString("no_evidence\n")
	} else {
		for i, item := range req.Pack.EvidenceItems {
			fmt.Fprintf(&b, "cite[%d] id=%s class=%s trust=%s surface=%q\n",
				i, item.ID, item.Class, item.TrustLevel, truncate(item.Surface, 160))
		}
	}
	text := b.String()
	sum := sha256.Sum256([]byte(text + string(req.Pack.Checksum)))
	call := models.ModelCall{
		ID:            ids.ModelCallID("mc_" + hex.EncodeToString(sum[:8])),
		ProjectID:     req.ProjectID,
		PackID:        req.Pack.ID,
		ProviderID:    ProviderID,
		ModelVersion:  ModelVersion,
		InputChecksum: hex.EncodeToString(sum[:]),
		Status:        "completed",
	}
	return models.CompletionResult{Text: text, ModelCall: call}, nil
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var _ models.Completer = Completer{}
