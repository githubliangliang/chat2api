package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyProviderCostPreservesUserMultiplier(t *testing.T) {
	cost := &CostBreakdown{TotalCost: 0.001816, ActualCost: 0.002724}

	applyProviderCost(cost, 0.000608736, 1.5)

	require.InDelta(t, 0.000608736, cost.TotalCost, 1e-12)
	require.InDelta(t, 0.000913104, cost.ActualCost, 1e-12)
}
