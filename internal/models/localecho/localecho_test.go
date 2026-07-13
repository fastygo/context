package localecho_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/models/localecho"
	"github.com/fastygo/context/internal/retrieval"
)

func TestLocalEchoCitesEvidence(t *testing.T) {
	out, err := (localecho.Completer{}).Complete(context.Background(), models.CompletionRequest{
		ProjectID: "p1",
		Pack: retrieval.ContextPack{
			ID: "pack1", ProjectID: "p1", RetrievalPlanID: "plan1",
			Checksum: "deadbeef",
			EvidenceItems: []retrieval.EvidenceItem{{
				ID: "e1", Class: foundation.EvidenceSourceText, TrustLevel: foundation.TrustProject,
				Surface: "New Year promo",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ModelCall.ProviderID != localecho.ProviderID || out.ModelCall.ModelVersion != localecho.ModelVersion {
		t.Fatalf("%#v", out.ModelCall)
	}
	if !strings.Contains(out.Text, "cite[0]") || !strings.Contains(out.Text, "New Year promo") {
		t.Fatalf("text=%q", out.Text)
	}
}
