package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAccountModelPlazaRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/model-plaza", handler.ListAccountModelPlaza)
	return router
}

func TestListAccountModelPlaza_AggregatesActiveAccounts(t *testing.T) {
	svc := newStubAdminService()
	svc.accounts = []service.Account{
		{
			ID:       1,
			Name:     "yj",
			Platform: service.PlatformGrok,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"grok-4.5": "grok-4.5"},
			},
		},
		{
			ID:       2,
			Name:     "shu26",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5.4": "gpt-5.4",
					"gpt-5.5": "gpt-5.5",
				},
			},
		},
		{
			ID:       3,
			Name:     "tabitoken",
			Platform: service.PlatformAnthropic,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusError,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"claude-opus-5": "claude-opus-5"},
			},
		},
	}

	router := setupAccountModelPlazaRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-plaza", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []service.AccountModelPlazaItem `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, []service.AccountModelPlazaItem{
		{ClientID: "grok-4.5", Platform: service.PlatformGrok, UpstreamIDs: []string{"grok-4.5"}, AccountCount: 1},
		{ClientID: "gpt-5.4", Platform: service.PlatformOpenAI, UpstreamIDs: []string{"gpt-5.4"}, AccountCount: 1},
		{ClientID: "gpt-5.5", Platform: service.PlatformOpenAI, UpstreamIDs: []string{"gpt-5.5"}, AccountCount: 1},
	}, body.Data.Items)
}

func TestListAccountModelPlaza_EmptyWhenNoActiveAccounts(t *testing.T) {
	svc := newStubAdminService()
	svc.accounts = []service.Account{
		{ID: 3, Platform: service.PlatformAnthropic, Status: service.StatusError},
	}

	router := setupAccountModelPlazaRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-plaza", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data struct {
			Items []service.AccountModelPlazaItem `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Empty(t, body.Data.Items)
}
