//go:build unit

package service

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPartialOpenAIStreamResultPreservesObservedUsageOnNonFailoverError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("X-Request-ID", "upstream-req-1")
	start := time.Now().Add(-250 * time.Millisecond)
	stream := &openaiStreamingResult{
		usage:        &OpenAIUsage{InputTokens: 12, OutputTokens: 3},
		firstTokenMs: partialIntPtr(42),
		responseID:   "resp_partial_1",
	}

	got := partialOpenAIStreamResult(c, resp, &Account{ID: 30}, stream, "gpt-5.6-sol", "gpt-5.6-sol", start, errors.New("stream usage incomplete: missing terminal event"))

	require.NotNil(t, got)
	require.Equal(t, 12, got.Usage.InputTokens)
	require.Equal(t, 3, got.Usage.OutputTokens)
	require.Equal(t, "upstream-req-1", got.RequestID)
	require.Equal(t, "resp_partial_1", got.ResponseID)
	require.True(t, got.Stream)
	require.Equal(t, 42, *got.FirstTokenMs)
}

func TestPartialOpenAIStreamResultDropsUsageForFailoverError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	stream := &openaiStreamingResult{usage: &OpenAIUsage{InputTokens: 12}}

	got := partialOpenAIStreamResult(c, &http.Response{}, &Account{ID: 30}, stream, "model", "model", time.Now(), &UpstreamFailoverError{StatusCode: http.StatusBadGateway})

	require.Nil(t, got)
}

func partialIntPtr(v int) *int { return &v }
