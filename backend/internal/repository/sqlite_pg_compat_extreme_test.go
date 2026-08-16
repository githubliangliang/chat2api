package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// GREATEST/LEAST 的 shim 原先只处理数值：首个参数非数值时直接原样返回，
// 完全不比较。ops_ingress_reject_repo.go 正是拿它比较时间戳，后果是聚合行的
// last_seen 冻结在首次写入的时刻，新事件的时间被丢弃（同一 key 第二次命中才显形）。
func newPGCompatSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	registerSQLitePGCompatFunctions()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	return db
}

func TestGreatestLeastAdvanceTimestampAggregates(t *testing.T) {
	db := newPGCompatSQLiteDB(t)
	_, err := db.Exec(`CREATE TABLE agg (
		k TEXT PRIMARY KEY, first_seen DATETIME, last_seen DATETIME, n INTEGER)`)
	require.NoError(t, err)

	early := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	late := early.Add(2 * time.Hour)
	_, err = db.Exec(`INSERT INTO agg VALUES ('a', ?, ?, 1)`, early, early)
	require.NoError(t, err)

	// 与 ops_ingress_reject_repo.go 的 upsert 写法一致
	_, err = db.Exec(`INSERT INTO agg VALUES ('a', ?, ?, 1)
		ON CONFLICT (k) DO UPDATE SET
			n = agg.n + EXCLUDED.n,
			first_seen = LEAST(agg.first_seen, EXCLUDED.first_seen),
			last_seen  = GREATEST(agg.last_seen, EXCLUDED.last_seen)`, late, late)
	require.NoError(t, err)

	var first, last time.Time
	var n int
	require.NoError(t, db.QueryRow(`SELECT first_seen, last_seen, n FROM agg WHERE k='a'`).
		Scan(&first, &last, &n))
	require.Equal(t, 2, n)
	require.True(t, first.Equal(early), "LEAST 必须保留最早时间，得到 %s", first)
	require.True(t, last.Equal(late), "GREATEST 必须推进到最新时间，得到 %s", last)

	// 反向：新事件更早时，last_seen 不应倒退
	earlier := early.Add(-time.Hour)
	_, err = db.Exec(`INSERT INTO agg VALUES ('a', ?, ?, 1)
		ON CONFLICT (k) DO UPDATE SET
			first_seen = LEAST(agg.first_seen, EXCLUDED.first_seen),
			last_seen  = GREATEST(agg.last_seen, EXCLUDED.last_seen)`, earlier, earlier)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT first_seen, last_seen FROM agg WHERE k='a'`).
		Scan(&first, &last))
	require.True(t, first.Equal(earlier), "LEAST 必须接受更早的时间，得到 %s", first)
	require.True(t, last.Equal(late), "GREATEST 不能因更早的事件而倒退，得到 %s", last)
}

func TestGreatestLeastNumericBehaviourUnchanged(t *testing.T) {
	db := newPGCompatSQLiteDB(t)
	var g, l int64
	require.NoError(t, db.QueryRow(`SELECT GREATEST(3,7,5), LEAST(3,7,5)`).Scan(&g, &l))
	require.EqualValues(t, 7, g)
	require.EqualValues(t, 3, l)

	// 余额/并发钳位（user_repo.go 的真实用法）保持整数返回
	var clamped int64
	require.NoError(t, db.QueryRow(`SELECT GREATEST(-5 + 2, 0)`).Scan(&clamped))
	require.EqualValues(t, 0, clamped)

	var f float64
	require.NoError(t, db.QueryRow(`SELECT LEAST(1.5, 2.5)`).Scan(&f))
	require.InDelta(t, 1.5, f, 1e-9)
}

func TestGreatestLeastIgnoresNullLikePostgres(t *testing.T) {
	db := newPGCompatSQLiteDB(t)
	var got sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT GREATEST(NULL, 4, NULL)`).Scan(&got))
	require.True(t, got.Valid, "只要有非 NULL 操作数就不该返回 NULL")
	require.EqualValues(t, 4, got.Int64)

	require.NoError(t, db.QueryRow(`SELECT LEAST(NULL, NULL)`).Scan(&got))
	require.False(t, got.Valid, "全为 NULL 时返回 NULL")
}
