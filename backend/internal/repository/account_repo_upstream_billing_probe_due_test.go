package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseSQLiteTimestampAcceptsStoredEntTime(t *testing.T) {
	want := time.Date(2026, 8, 13, 12, 34, 56, 123000000, time.UTC)
	got, ok := parseSQLiteTimestamp(want.Format("2006-01-02 15:04:05.999999999 -0700 MST"))
	require.True(t, ok)
	require.Equal(t, want, got)
}
