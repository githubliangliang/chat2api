package service

import "math"

// applyProviderCost replaces the local estimate with an authoritative
// per-request provider charge while retaining the configured user multiplier.
// The estimated line items are scaled proportionally so their sum remains the
// same as TotalCost in usage-log detail views.
func applyProviderCost(cost *CostBreakdown, providerCostUSD, rateMultiplier float64) {
	if cost == nil || math.IsNaN(providerCostUSD) || math.IsInf(providerCostUSD, 0) || providerCostUSD < 0 {
		return
	}
	if cost.TotalCost > 0 && !math.IsNaN(cost.TotalCost) && !math.IsInf(cost.TotalCost, 0) {
		scale := providerCostUSD / cost.TotalCost
		cost.InputCost *= scale
		cost.ImageInputCost *= scale
		cost.OutputCost *= scale
		cost.ImageOutputCost *= scale
		cost.CacheCreationCost *= scale
		cost.CacheReadCost *= scale
	} else {
		cost.InputCost = 0
		cost.ImageInputCost = 0
		cost.OutputCost = 0
		cost.ImageOutputCost = 0
		cost.CacheCreationCost = 0
		cost.CacheReadCost = 0
	}
	cost.TotalCost = providerCostUSD
	cost.ActualCost = providerCostUSD * rateMultiplier
}
