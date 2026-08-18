package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func ollamaCloudUsageRepositoryAccount() *service.Account {
	return &service.Account{
		ID: 17, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "key", "base_url": "https://ollama.com"},
		Extra: map[string]any{
			service.OllamaCloudUsageSessionExtraKey:     "cipher:wos-session=secret",
			service.OllamaCloudUsageAutoRefreshExtraKey: true,
		},
	}
}

func TestOllamaCloudBaseURLSQLRegexMatchesServiceSemantics(t *testing.T) {
	for _, baseURL := range []string{
		"https://ollama.com",
		"HTTPS://WWW.OLLAMA.COM:443/v1",
		"https://ollama.com/V1",
		"https://ollama.com/v1/",
		"https://ollama.com.evil.test/v1",
	} {
		t.Run(baseURL, func(t *testing.T) {
			matched, err := regexp.MatchString(ollamaCloudBaseURLRegexSQL, baseURL)
			require.NoError(t, err)
			account := ollamaCloudUsageRepositoryAccount()
			account.Credentials["base_url"] = baseURL
			require.Equal(t, service.IsOllamaCloudUsageAccount(account), matched)
		})
	}
}

func TestOllamaCloudUsageManagedPayloadCopiesOnlyManagedFields(t *testing.T) {
	account := ollamaCloudUsageRepositoryAccount()
	account.Extra["unmanaged"] = "value"
	payload := ollamaCloudUsageManagedPayload(account)
	require.Equal(t, "cipher:wos-session=secret", payload[service.OllamaCloudUsageSessionExtraKey])
	require.Equal(t, true, payload[service.OllamaCloudUsageAutoRefreshExtraKey])
	require.NotContains(t, payload, "unmanaged")
}

func TestOllamaCloudUsageSessionRequiredGuards(t *testing.T) {
	repo := &accountRepository{}
	account := ollamaCloudUsageRepositoryAccount()
	delete(account.Extra, service.OllamaCloudUsageSessionExtraKey)
	require.ErrorIs(t, repo.SetOllamaCloudUsageAutoRefresh(context.TODO(), account, true), service.ErrOllamaCloudUsageSessionRequired)
	require.ErrorIs(t, repo.DisableOllamaCloudUsageAutoRefresh(context.TODO(), account), service.ErrOllamaCloudUsageSessionRequired)
}
