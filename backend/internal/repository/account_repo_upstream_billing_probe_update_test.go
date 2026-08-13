package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestProbeExtraUpdatePredicates(t *testing.T) {
	require.True(t, upstreamBillingProbeExplicitlyDisabled(map[string]any{service.UpstreamBillingProbeEnabledExtraKey: false}))
	require.True(t, upstreamBillingProbeSnapshotClearRequested(map[string]any{service.UpstreamBillingProbeExtraKey: nil}))
	require.False(t, upstreamBillingProbeExplicitlyDisabled(map[string]any{service.UpstreamBillingProbeEnabledExtraKey: true}))
}
