package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONValuesEqualUsesCanonicalObjectOrder(t *testing.T) {
	require.True(t, jsonValuesEqual(map[string]any{"a": 1.0, "b": true}, map[string]any{"b": true, "a": 1.0}))
	require.False(t, jsonValuesEqual(map[string]any{"a": 1.0}, map[string]any{"a": 2.0}))
}
