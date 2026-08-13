package repository

import (
	"fmt"
	"strings"
)

// sqlSliceIn builds a portable "col IN ($n,$n+1,...)" clause for SQLite
// and PostgreSQL from any scalar slice.
func sqlSliceIn[T any](column string, values []T, startArg int) (clause string, args []any) {
	if len(values) == 0 {
		return "1=0", nil
	}
	parts := make([]string, len(values))
	args = make([]any, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("$%d", startArg+i)
		args[i] = value
	}
	return fmt.Sprintf("%s IN (%s)", column, strings.Join(parts, ",")), args
}

// sqlInt64In builds a portable "col IN ($n,$n+1,...)" clause for SQLite and PostgreSQL.
// startArg is the first placeholder index (1-based, matching $1 style).
func sqlInt64In(column string, ids []int64, startArg int) (clause string, args []any) {
	return sqlSliceIn(column, ids, startArg)
}
