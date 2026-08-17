package service

import (
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func TestAggregateAccountModelPlaza_Empty(t *testing.T) {
	require.Empty(t, AggregateAccountModelPlaza(nil))
	require.Empty(t, AggregateAccountModelPlaza([]Account{}))
}

func TestAggregateAccountModelPlaza_ExcludesNonActive(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:       3,
			Name:     "tabitoken",
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			Status:   StatusError,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-opus-4-8": "claude-opus-4-8",
					"claude-opus-5":   "claude-opus-5",
				},
			},
		},
	})
	require.Empty(t, items)
}

func TestAggregateAccountModelPlaza_MappingKeysAndUpstreamValues(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:       2,
			Name:     "shu26",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5.4":      "gpt-5.4",
					"gpt-image-2":  "gpt-image-2",
					"alias-review": "codex-auto-review",
				},
			},
		},
	})
	require.Equal(t, []AccountModelPlazaItem{
		{ClientID: "alias-review", Platform: PlatformOpenAI, UpstreamIDs: []string{"codex-auto-review"}, AccountCount: 1},
		{ClientID: "gpt-5.4", Platform: PlatformOpenAI, UpstreamIDs: []string{"gpt-5.4"}, AccountCount: 1},
		{ClientID: "gpt-image-2", Platform: PlatformOpenAI, UpstreamIDs: []string{"gpt-image-2"}, AccountCount: 1},
	}, items)
}

func TestAggregateAccountModelPlaza_SameClientIDTwoPlatforms(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:       1,
			Platform: PlatformGrok,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"shared-model": "grok-upstream"},
			},
		},
		{
			ID:       2,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"shared-model": "openai-upstream"},
			},
		},
	})
	require.Equal(t, []AccountModelPlazaItem{
		{ClientID: "shared-model", Platform: PlatformGrok, UpstreamIDs: []string{"grok-upstream"}, AccountCount: 1},
		{ClientID: "shared-model", Platform: PlatformOpenAI, UpstreamIDs: []string{"openai-upstream"}, AccountCount: 1},
	}, items)
}

func TestAggregateAccountModelPlaza_DedupesAccountsAndUpstreamIDs(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:       1,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
			},
		},
		{
			ID:       2,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4-20260301"},
			},
		},
	})
	require.Equal(t, []AccountModelPlazaItem{
		{
			ClientID:     "gpt-5.4",
			Platform:     PlatformOpenAI,
			UpstreamIDs:  []string{"gpt-5.4", "gpt-5.4-20260301"},
			AccountCount: 2,
		},
	}, items)
}

func TestAggregateAccountModelPlaza_OpenAIPassthroughUsesDefaults(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:       9,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"custom-only": "custom-only"},
			},
			Extra: map[string]any{"openai_passthrough": true},
		},
	})
	require.Equal(t, sortedCopy(openai.DefaultModelIDs()), clientIDs(items))
	for _, item := range items {
		require.Equal(t, []string{item.ClientID}, item.UpstreamIDs)
		require.Equal(t, 1, item.AccountCount)
	}
}

func TestAggregateAccountModelPlaza_EmptyMappingUsesPlatformDefaults(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:          1,
			Platform:    PlatformGrok,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Credentials: map[string]any{},
		},
	})
	require.Equal(t, sortedCopy(xai.DefaultModelIDs()), clientIDs(items))
}

func TestAggregateAccountModelPlaza_IncludesRateLimitedActive(t *testing.T) {
	items := AggregateAccountModelPlaza([]Account{
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: false,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
			},
		},
	})
	require.Equal(t, []AccountModelPlazaItem{
		{ClientID: "gpt-5.4", Platform: PlatformOpenAI, UpstreamIDs: []string{"gpt-5.4"}, AccountCount: 1},
	}, items)
}

func clientIDs(items []AccountModelPlazaItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ClientID
	}
	return out
}

func sortedCopy(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}
