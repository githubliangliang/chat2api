package repository

import (
	"fmt"
	"strings"
)

// sqlInt64In builds a portable "col IN ($n,$n+1,...)" clause for SQLite and PostgreSQL.
// startArg is the first placeholder index (1-based, matching $1 style).
func sqlInt64In(column string, ids []int64, startArg int) (clause string, args []any) {
	if len(ids) == 0 {
		// Always-false predicate keeps SQL valid without special-casing callers.
		return "1=0", nil
	}
	parts := make([]string, len(ids))
	args = make([]any, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("$%d", startArg+i)
		args[i] = id
	}
	return fmt.Sprintf("%s IN (%s)", column, strings.Join(parts, ",")), args
}
