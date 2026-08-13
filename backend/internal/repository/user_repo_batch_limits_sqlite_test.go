package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryBatchLimitsSQLite(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()
	for _, input := range []*service.User{
		{Email: "one@example.com", Username: "one", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Concurrency: 1, RPMLimit: 10},
		{Email: "two@example.com", Username: "two", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Concurrency: 2, RPMLimit: 20},
		{Email: "three@example.com", Username: "three", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Concurrency: 3, RPMLimit: 30},
	} {
		require.NoError(t, repo.Create(ctx, input))
	}
	users, err := client.User.Query().Order(user.ByID()).All(ctx)
	require.NoError(t, err)
	require.Len(t, users, 3)

	affected, err := repo.BatchSetConcurrency(ctx, []int64{users[0].ID, users[1].ID}, 7)
	require.NoError(t, err)
	require.Equal(t, 2, affected)

	affected, err = repo.BatchAddConcurrency(ctx, []int64{users[0].ID, users[2].ID}, -20)
	require.NoError(t, err)
	require.Equal(t, 2, affected)

	concurrency, rpm := 4, 44
	affected, err = repo.BatchUpdateLimits(ctx, []int64{users[1].ID, users[2].ID}, &concurrency, &rpm)
	require.NoError(t, err)
	require.Equal(t, 2, affected)

	got, err := client.User.Query().Order(user.ByID()).All(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, got[0].Concurrency)
	require.Equal(t, 4, got[1].Concurrency)
	require.Equal(t, 4, got[2].Concurrency)
	require.Equal(t, 10, got[0].RpmLimit)
	require.Equal(t, 44, got[1].RpmLimit)
	require.Equal(t, 44, got[2].RpmLimit)
}

func TestUserRepositoryAttributeFilterSQLiteIsCaseInsensitive(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()
	input := &service.User{Email: "attr@example.com", Username: "attr-user", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, repo.Create(ctx, input))
	created, err := client.User.Query().Only(ctx)
	require.NoError(t, err)
	definition, err := client.UserAttributeDefinition.Create().SetKey("tier").SetName("Tier").SetType("text").Save(ctx)
	require.NoError(t, err)

	_, err = repo.sql.ExecContext(ctx, `INSERT INTO user_attribute_values (user_id, attribute_id, value, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, created.ID, definition.ID, "Premium-Customer")
	require.NoError(t, err)

	ids, err := repo.filterUsersByAttributes(ctx, map[int64]string{definition.ID: "premium"})
	require.NoError(t, err)
	require.Equal(t, []int64{created.ID}, ids)
}

func TestUserRepositoryDeductAvailableBalanceSQLite(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()
	input := &service.User{Email: "balance@example.com", Username: "balance-user", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Balance: 5}
	require.NoError(t, repo.Create(ctx, input))
	created, err := repo.client.User.Query().Only(ctx)
	require.NoError(t, err)

	deducted, err := repo.DeductAvailableBalance(ctx, created.ID, 10)
	require.NoError(t, err)
	require.Equal(t, 5.0, deducted)
}
