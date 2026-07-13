package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fastygo/context/internal/policy/quota"
)

// LoadQuotaLimitsFromEnv reads soft project quotas (Chunk 28 / ADR-0025).
// Zero (unset) means unlimited. No billing.
//
//	CONTEXT_QUOTA_MAX_CHUNKS
//	CONTEXT_QUOTA_MAX_PACKS
//	CONTEXT_QUOTA_MAX_RUNS
//	CONTEXT_QUOTA_SOFT_ASK_PERCENT  (default 80 when any max is set)
func LoadQuotaLimitsFromEnv() (quota.Limits, error) {
	var lim quota.Limits
	var err error
	if lim.MaxChunks, err = envIntNonNeg("CONTEXT_QUOTA_MAX_CHUNKS"); err != nil {
		return quota.Limits{}, err
	}
	if lim.MaxPacks, err = envIntNonNeg("CONTEXT_QUOTA_MAX_PACKS"); err != nil {
		return quota.Limits{}, err
	}
	if lim.MaxRuns, err = envIntNonNeg("CONTEXT_QUOTA_MAX_RUNS"); err != nil {
		return quota.Limits{}, err
	}
	if lim.SoftAskPercent, err = envIntNonNeg("CONTEXT_QUOTA_SOFT_ASK_PERCENT"); err != nil {
		return quota.Limits{}, err
	}
	return lim, nil
}

func envIntNonNeg(key string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s: must be >= 0", key)
	}
	return n, nil
}
