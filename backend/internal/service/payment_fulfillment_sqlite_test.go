package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAffiliateRebateAuditClaimQueryUsesSQLiteSyntax(t *testing.T) {
	query, args := buildAffiliateRebateAuditClaimQuery(nil, "42", `{"status":"reserved"}`)
	require.NotContains(t, query, "::")
	require.NotContains(t, query, "NOW()")
	require.Contains(t, query, "CURRENT_TIMESTAMP")
	require.Equal(t, []any{"42", `{"status":"reserved"}`, "42"}, args)
}
