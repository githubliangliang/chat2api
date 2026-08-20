package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIResponseServiceTierPrefersProviderDeclaration(t *testing.T) {
	got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"service_tier":"default","usage":{"input_tokens":1}}`))
	if got == nil || *got != "default" {
		t.Fatalf("service tier = %v, want default", got)
	}
}

func TestExtractOpenAIResponseServiceTierReadsResponsesEnvelope(t *testing.T) {
	got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"type":"response.completed","response":{"service_tier":"flex","usage":{"input_tokens":1}}}`))
	if got == nil || *got != "flex" {
		t.Fatalf("service tier = %v, want flex", got)
	}
}

func TestExtractOpenAIResponseServiceTierReturnsNilWhenAbsent(t *testing.T) {
	if got := extractOpenAIResponseServiceTierFromJSONBytes([]byte(`{"usage":{"input_tokens":1}}`)); got != nil {
		t.Fatalf("service tier = %v, want nil", got)
	}
}

func TestLogOpenAIServiceTierDifferenceIncludesReconciliationFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logSink, restore := captureStructuredLog(t)
	defer restore()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	requested, provider := "priority", "default"
	logOpenAIServiceTierDifference(c, &Account{ID: 917}, "rid-tier", &requested, &provider)

	require.True(t, logSink.ContainsMessageAtLevel("openai service tier differs from request", "info"))
	require.True(t, logSink.ContainsFieldValue("request_id", "rid-tier"))
	require.True(t, logSink.ContainsFieldValue("account_id", "917"))
	require.True(t, logSink.ContainsFieldValue("requested_service_tier", "priority"))
	require.True(t, logSink.ContainsFieldValue("provider_service_tier", "default"))
	require.True(t, logSink.ContainsFieldValue("final_service_tier", "default"))
}

func TestLogOpenAIServiceTierDifferenceSkipsEqualOrMissingProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logSink, restore := captureStructuredLog(t)
	defer restore()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	tier := "flex"
	logOpenAIServiceTierDifference(c, &Account{ID: 917}, "rid-equal", &tier, &tier)
	logOpenAIServiceTierDifference(c, &Account{ID: 917}, "rid-missing", &tier, nil)

	require.False(t, logSink.ContainsMessageAtLevel("openai service tier differs from request", "info"))
}
