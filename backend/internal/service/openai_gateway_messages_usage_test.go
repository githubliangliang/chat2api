//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestCopyOpenAIUsageFromResponsesUsageTrustsCanonicalCacheCreationValue(t *testing.T) {
	usage := &apicompat.ResponsesUsage{
		InputTokens:              20,
		OutputTokens:             2,
		CacheCreationInputTokens: 0,
		InputTokensDetails: &apicompat.ResponsesInputTokensDetails{
			CachedTokens:     3,
			CacheWriteTokens: 19,
		},
	}

	got := copyOpenAIUsageFromResponsesUsage(usage)

	require.Equal(t, 20, got.InputTokens)
	require.Equal(t, 3, got.CacheReadInputTokens)
	require.Zero(t, got.CacheCreationInputTokens)
}

func TestCopyOpenAIUsageFromResponsesUsagePreservesProviderCost(t *testing.T) {
	costTicks := int64(6_087_360)
	usage := &apicompat.ResponsesUsage{
		InputTokens:    212,
		OutputTokens:   264,
		CostInUSDTicks: &costTicks,
	}

	got := copyOpenAIUsageFromResponsesUsage(usage)

	require.NotNil(t, got.ProviderCostUSD)
	require.InDelta(t, 0.000608736, *got.ProviderCostUSD, 1e-12)
}
