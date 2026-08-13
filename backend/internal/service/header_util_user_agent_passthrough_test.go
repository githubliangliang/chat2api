package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyClientUserAgentPassthrough_WireAndCanonical(t *testing.T) {
	client := http.Header{}
	client.Set("User-Agent", "my-client/1.2.3 (custom)")

	dstWire := http.Header{}
	dstWire.Set("User-Agent", "claude-cli/9.9.9 (external, cli)")
	applyClientUserAgentPassthrough(dstWire, client, true)
	require.Equal(t, "my-client/1.2.3 (custom)", getHeaderRaw(dstWire, "User-Agent"))

	dstSet := http.Header{}
	dstSet.Set("User-Agent", "codex_cli_rs/0.1.0")
	applyClientUserAgentPassthrough(dstSet, client, false)
	require.Equal(t, "my-client/1.2.3 (custom)", dstSet.Get("User-Agent"))
}

func TestApplyClientUserAgentPassthrough_EmptyClientKeepsExisting(t *testing.T) {
	dst := http.Header{}
	dst.Set("User-Agent", "keep-me")
	applyClientUserAgentPassthrough(dst, http.Header{}, true)
	require.Equal(t, "keep-me", getHeaderRaw(dst, "User-Agent"))
}
