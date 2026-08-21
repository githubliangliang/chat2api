package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

// GetStatsWithFilters 的总计和三个端点分项必须套同一组筛选条件。
// upstream_model_mismatch 此前只加在总计 SQL 上，端点/上游端点/路径三个分项的
// 查询根本没有这个参数，于是管理端按「上游模型不一致」筛选时页面自相矛盾：
// 总计变了、分项没变。
func TestUsageLogRepositoryGetStatsWithFiltersAppliesUpstreamModelMismatchToEndpointBreakdowns(t *testing.T) {
	summaryColumns := []string{
		"total_requests",
		"total_input_tokens",
		"total_output_tokens",
		"total_cache_tokens",
		"total_cache_creation_tokens",
		"total_cache_read_tokens",
		"total_cost",
		"total_actual_cost",
		"total_account_cost",
		"avg_duration_ms",
	}
	endpointColumns := []string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}

	tests := []struct {
		name      string
		mismatch  bool
		condition string
	}{
		{name: "mismatch only", mismatch: true, condition: "upstream_model_mismatch IS TRUE"},
		{name: "matched only", mismatch: false, condition: "upstream_model_mismatch IS FALSE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}

			mismatch := tt.mismatch
			filters := usagestats.UsageLogFilters{UpstreamModelMismatch: &mismatch}

			mock.ExpectQuery("(?s)FROM usage_logs.*" + tt.condition).
				WillReturnRows(sqlmock.NewRows(summaryColumns).
					AddRow(int64(1), int64(2), int64(3), int64(4), int64(1), int64(3), 1.2, 1.0, 1.2, 20.0))
			mock.ExpectQuery("(?s)COALESCE\\(NULLIF\\(TRIM\\(inbound_endpoint\\), ''\\), 'unknown'\\) AS endpoint.*"+tt.condition+" GROUP BY endpoint").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows(endpointColumns))
			mock.ExpectQuery("(?s)COALESCE\\(NULLIF\\(TRIM\\(upstream_endpoint\\), ''\\), 'unknown'\\) AS endpoint.*"+tt.condition+" GROUP BY endpoint").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows(endpointColumns))
			mock.ExpectQuery("(?s)SELECT\\s+CONCAT\\(.*"+tt.condition+" GROUP BY endpoint").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows(endpointColumns))

			stats, err := repo.GetStatsWithFilters(context.Background(), filters)
			require.NoError(t, err)
			require.Equal(t, int64(1), stats.TotalRequests)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// 不传筛选时四条 SQL 里都不得出现该条件，避免默认视图被悄悄收窄。
// Go 的 regexp 是 RE2，没有负向断言，所以这里换成记录实际 SQL 再断言。
func TestUsageLogRepositoryGetStatsWithFiltersOmitsUpstreamModelMismatchWhenUnset(t *testing.T) {
	var executed []string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(
		sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
			executed = append(executed, actualSQL)
			return nil
		}),
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &usageLogRepository{sql: db}

	mock.ExpectQuery("").
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests", "total_input_tokens", "total_output_tokens", "total_cache_tokens",
			"total_cache_creation_tokens", "total_cache_read_tokens", "total_cost",
			"total_actual_cost", "total_account_cost", "avg_duration_ms",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), 0.0, 0.0, 0.0, 0.0))
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("").
			WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	}

	_, err = repo.GetStatsWithFilters(context.Background(), usagestats.UsageLogFilters{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, executed, 4, "summary + inbound/upstream endpoints + endpoint paths")
	for i, query := range executed {
		require.NotContains(t, query, "upstream_model_mismatch", "query %d must stay unfiltered", i)
	}
}
