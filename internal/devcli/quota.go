package devcli

import (
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/policy/quota"
)

// QuotaResult is CLI/HTTP JSON for quota status.
type QuotaResult = quota.Status

func usageFromState(st State) quota.Usage {
	return quota.Usage{
		Chunks: len(st.Chunks),
		Packs:  len(st.Packs),
		Runs:   len(st.Runs),
	}
}

// QuotaStatus evaluates soft project quotas against workspace counters.
func QuotaStatus(dataDir string) (QuotaResult, error) {
	if dataDir == "" {
		return QuotaResult{}, apperr.New(apperr.Validation, "data dir required")
	}
	limits, err := config.LoadQuotaLimitsFromEnv()
	if err != nil {
		return QuotaResult{}, err
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return QuotaResult{}, err
	}
	return quota.Evaluate(limits, usageFromState(st)), nil
}

// requireQuotaResource blocks mutating work when the resource is at hard limit.
func requireQuotaResource(dataDir, resource string) error {
	limits, err := config.LoadQuotaLimitsFromEnv()
	if err != nil {
		return err
	}
	if !limits.Enabled() {
		return nil
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return err
	}
	usage := usageFromState(st)
	if !quota.BlocksResource(limits, usage, resource) {
		return nil
	}
	stq := quota.Evaluate(limits, usage)
	msg := fmt.Sprintf("quota deny: %s at hard limit", resource)
	for _, b := range stq.Breaches {
		if b.Resource == resource && b.Decision == policy.DecisionDeny {
			msg = "quota deny: " + b.Reason
			break
		}
	}
	return apperr.New(apperr.Permission, msg)
}
