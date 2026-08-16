package repository

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	sqlite "modernc.org/sqlite"
)

var registerSQLitePGCompatOnce sync.Once

// registerSQLitePGCompatFunctions registers a few PostgreSQL-shaped helpers so
// existing raw SQL (NOW/GREATEST/LEAST/TO_CHAR) can run on SQLite for personal deploys.
// This is best-effort compatibility, not full PostgreSQL parity.
func registerSQLitePGCompatFunctions() {
	registerSQLitePGCompatOnce.Do(func() {
		// NOW() — used widely in repository SQL for updated_at stamps.
		_ = sqlite.RegisterScalarFunction("now", 0, func(_ *sqlite.FunctionContext, _ []driver.Value) (driver.Value, error) {
			return time.Now().UTC().Format(time.RFC3339Nano), nil
		})
		// GREATEST/LEAST — multi-arg numeric helpers (nArg=-1).
		_ = sqlite.RegisterScalarFunction("greatest", -1, sqliteNumericExtreme(true))
		_ = sqlite.RegisterScalarFunction("least", -1, sqliteNumericExtreme(false))
		// TO_CHAR(ts, format) — usage trend/stats bucketing (hour/day/week/month).
		_ = sqlite.RegisterScalarFunction("to_char", 2, sqliteToChar)
		// HOST(inet) — PG inet helper; SQLite stores client_ip as TEXT so return as-is.
		_ = sqlite.RegisterScalarFunction("host", 1, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			if len(args) == 0 || args[0] == nil {
				return nil, nil
			}
			switch v := args[0].(type) {
			case string:
				return v, nil
			case []byte:
				return string(v), nil
			default:
				return fmt.Sprint(v), nil
			}
		})
	})
}

// sqliteToChar implements a minimal PostgreSQL TO_CHAR for usage analytics formats.
func sqliteToChar(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	if len(args) < 2 || args[0] == nil {
		return nil, nil
	}
	ts, ok := parseSQLiteTime(args[0])
	if !ok {
		return nil, fmt.Errorf("to_char: unsupported timestamp %T", args[0])
	}
	format, _ := args[1].(string)
	switch strings.TrimSpace(format) {
	case "YYYY-MM-DD HH24:00":
		return ts.UTC().Format("2006-01-02 15:00"), nil
	case "YYYY-MM-DD":
		return ts.UTC().Format("2006-01-02"), nil
	case "YYYY-MM":
		return ts.UTC().Format("2006-01"), nil
	case "IYYY-IW":
		year, week := ts.UTC().ISOWeek()
		return fmt.Sprintf("%04d-%02d", year, week), nil
	default:
		// Best-effort: map a few common PG tokens to Go layouts.
		layout := strings.NewReplacer(
			"YYYY", "2006",
			"MM", "01",
			"DD", "02",
			"HH24", "15",
			"MI", "04",
			"SS", "05",
		).Replace(format)
		return ts.UTC().Format(layout), nil
	}
}

func parseSQLiteTime(v driver.Value) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, false
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05-07",
			"2006-01-02 15:04:05.999999999Z07:00",
			"2006-01-02 15:04:05Z07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, s); err == nil {
				return parsed, true
			}
		}
		if len(s) >= 19 {
			if parsed, err := time.Parse("2006-01-02 15:04:05", s[:19]); err == nil {
				return parsed, true
			}
		}
		// unix seconds stored as text
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(n, 0).UTC(), true
		}
		return time.Time{}, false
	case int64:
		// Heuristic: ms vs seconds
		if t > 1_000_000_000_000 {
			return time.UnixMilli(t).UTC(), true
		}
		return time.Unix(t, 0).UTC(), true
	case float64:
		return time.Unix(int64(t), 0).UTC(), true
	default:
		return time.Time{}, false
	}
}

func sqliteNormalizedTimestampExpr(column string) string {
	return fmt.Sprintf(`CASE
		WHEN typeof(%[1]s) IN ('integer', 'real') THEN datetime(%[1]s, 'unixepoch')
		WHEN length(CAST(%[1]s AS TEXT)) >= 19 THEN datetime(substr(CAST(%[1]s AS TEXT), 1, 19))
		ELSE NULL
	END`, column)
}

func sqliteNumericExtreme(max bool) func(*sqlite.FunctionContext, []driver.Value) (driver.Value, error) {
	return func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("greatest/least requires at least 1 argument")
		}
		// PG 语义：忽略 NULL，全为 NULL 才返回 NULL。
		operands := make([]driver.Value, 0, len(args))
		for _, arg := range args {
			if arg != nil {
				operands = append(operands, arg)
			}
		}
		if len(operands) == 0 {
			return nil, nil
		}
		if result, ok := numericExtreme(max, operands); ok {
			return result, nil
		}
		// 非数值：时间戳与字符串同样需要比较。旧实现在这里直接返回
		// operands[0]，导致 GREATEST(last_seen, EXCLUDED.last_seen) 永远保留
		// 旧值——聚合行的 last_seen 会冻结在首次写入的时刻。
		return comparableExtreme(max, operands), nil
	}
}

// numericExtreme 在全部操作数均为数值时返回极值；只要有一个不是数值就放弃，
// 交给 comparableExtreme，避免把时间戳按 0 参与比较。
func numericExtreme(max bool, operands []driver.Value) (driver.Value, bool) {
	best, ok := asFloat64(operands[0])
	if !ok {
		return nil, false
	}
	for i := 1; i < len(operands); i++ {
		v, ok := asFloat64(operands[i])
		if !ok {
			return nil, false
		}
		if max == (v > best) && v != best {
			best = v
		}
	}
	// Prefer integer when all inputs were integers.
	if best == math.Trunc(best) {
		return int64(best), true
	}
	return best, true
}

// comparableExtreme 处理时间戳与字符串：能解析成时间的按时间序比较，
// 否则退化为字典序。返回的是原始操作数，保持列的存储形态不变。
func comparableExtreme(max bool, operands []driver.Value) driver.Value {
	bestIdx := 0
	bestTime, bestIsTime := parseSQLiteTime(operands[0])
	for i := 1; i < len(operands); i++ {
		candTime, candIsTime := parseSQLiteTime(operands[i])
		var newer bool
		switch {
		case bestIsTime && candIsTime:
			newer = candTime.After(bestTime)
		case bestIsTime != candIsTime:
			// 类型不可比时保持既有选择，不做猜测。
			continue
		default:
			newer = driverValueString(operands[i]) > driverValueString(operands[bestIdx])
		}
		if newer == max {
			bestIdx, bestTime, bestIsTime = i, candTime, candIsTime
		}
	}
	return operands[bestIdx]
}

func driverValueString(v driver.Value) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return fmt.Sprint(v)
	}
}

func asFloat64(v driver.Value) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}
