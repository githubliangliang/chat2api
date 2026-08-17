package service

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

// AccountModelPlazaItem is one deduped (client model ID, platform) row
// aggregated from enabled (status=active) accounts.
type AccountModelPlazaItem struct {
	ClientID     string   `json:"client_id"`
	Platform     string   `json:"platform"`
	UpstreamIDs  []string `json:"upstream_ids"`
	AccountCount int      `json:"account_count"`
}

type plazaAggKey struct {
	clientID string
	platform string
}

type plazaAggRow struct {
	upstream map[string]struct{}
	accounts map[int64]struct{}
}

// AggregateAccountModelPlaza expands enabled accounts the same way
// GET /admin/accounts/:id/models does, then dedupes by (client_id, platform).
// Non-active accounts are ignored even if the caller passed them.
func AggregateAccountModelPlaza(accounts []Account) []AccountModelPlazaItem {
	rows := make(map[plazaAggKey]*plazaAggRow)
	for i := range accounts {
		acc := &accounts[i]
		if acc == nil || acc.Status != StatusActive {
			continue
		}
		clientIDs := accountPlazaClientModelIDs(acc)
		if len(clientIDs) == 0 {
			continue
		}
		mapping := acc.GetModelMapping()
		platform := strings.TrimSpace(acc.Platform)
		for _, clientID := range clientIDs {
			clientID = strings.TrimSpace(clientID)
			if clientID == "" {
				continue
			}
			key := plazaAggKey{clientID: clientID, platform: platform}
			row := rows[key]
			if row == nil {
				row = &plazaAggRow{
					upstream: make(map[string]struct{}),
					accounts: make(map[int64]struct{}),
				}
				rows[key] = row
			}
			row.accounts[acc.ID] = struct{}{}
			upstream := strings.TrimSpace(mapping[clientID])
			if upstream == "" {
				upstream = clientID
			}
			row.upstream[upstream] = struct{}{}
		}
	}

	out := make([]AccountModelPlazaItem, 0, len(rows))
	for key, row := range rows {
		out = append(out, AccountModelPlazaItem{
			ClientID:     key.clientID,
			Platform:     key.platform,
			UpstreamIDs:  sortedStringSet(row.upstream),
			AccountCount: len(row.accounts),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return out[i].ClientID < out[j].ClientID
	})
	return out
}

// accountPlazaClientModelIDs mirrors AccountHandler.GetAvailableModels.
// Keep the two in sync: that endpoint is the per-account source of truth
// for which client IDs an account exposes.
func accountPlazaClientModelIDs(account *Account) []string {
	if account == nil {
		return nil
	}

	if account.IsOpenAI() {
		if account.IsOpenAIPassthroughEnabled() {
			return openai.DefaultModelIDs()
		}
		return mappingKeysOrDefaults(account.GetModelMapping(), openai.DefaultModelIDs())
	}

	if account.IsGemini() {
		if account.IsOAuth() {
			return geminiDefaultModelIDs()
		}
		return mappingKeysOrDefaults(account.GetModelMapping(), geminiDefaultModelIDs())
	}

	if account.Platform == PlatformAntigravity {
		return antigravityDefaultModelIDs()
	}

	if account.Platform == PlatformGrok {
		if !grokHasExplicitMapping(account) {
			return xai.DefaultModelIDs()
		}
		return mappingKeysOrDefaults(account.GetModelMapping(), xai.DefaultModelIDs())
	}

	if account.IsOAuth() {
		return claude.DefaultModelIDs()
	}
	return mappingKeysOrDefaults(account.GetModelMapping(), claude.DefaultModelIDs())
}

func mappingKeysOrDefaults(mapping map[string]string, defaults []string) []string {
	if len(mapping) == 0 {
		return defaults
	}
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keys = append(keys, trimmed)
		}
	}
	sort.Strings(keys)
	return keys
}

func grokHasExplicitMapping(account *Account) bool {
	if account == nil || account.Credentials == nil {
		return false
	}
	switch raw := account.Credentials["model_mapping"].(type) {
	case map[string]any:
		return len(raw) > 0
	case map[string]string:
		return len(raw) > 0
	default:
		return false
	}
}

func geminiDefaultModelIDs() []string {
	ids := make([]string, 0, len(geminicli.DefaultModels))
	for _, model := range geminicli.DefaultModels {
		ids = append(ids, model.ID)
	}
	return ids
}

func antigravityDefaultModelIDs() []string {
	models := antigravity.DefaultModels()
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func sortedStringSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for v := range values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
