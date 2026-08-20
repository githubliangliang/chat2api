package service

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestScanCCStreamCapturesProviderServiceTier(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"id":"chatcmpl_tier","object":"chat.completion.chunk","service_tier":"flex","choices":[]}`,
			"",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	scan := (&OpenAIGatewayService{}).scanCCStream(resp, "test", "rid-tier", time.Now(), func(*apicompat.ChatCompletionsChunk) {})

	require.NoError(t, scan.Err)
	require.Equal(t, "flex", scan.ServiceTier)
}
