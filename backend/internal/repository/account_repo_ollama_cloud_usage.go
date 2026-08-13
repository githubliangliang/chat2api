package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const ollamaCloudBaseURLRegexSQL = `^[hH][tT][tT][pP][sS]://([wW][wW][wW]\.)?[oO][lL][lL][aA][mM][aA]\.[cC][oO][mM](:443)?(/v1)?$`

const ollamaCloudUsageEligibleSQL = `
	platform IN ('openai', 'anthropic')
	AND type = 'apikey'
	AND lower(trim(json_extract(credentials, '$.base_url'))) IN (
		'https://ollama.com', 'https://ollama.com/v1', 'https://ollama.com:443',
		'https://ollama.com:443/v1', 'https://www.ollama.com', 'https://www.ollama.com/v1',
		'https://www.ollama.com:443', 'https://www.ollama.com:443/v1'
	)
	AND json_type(credentials, '$.api_key') = 'text'
`

func ollamaCloudBaseURLMatchesSQL(expression string) string {
	return "lower(trim(" + expression + ")) IN ('https://ollama.com', 'https://ollama.com/v1', 'https://ollama.com:443', 'https://ollama.com:443/v1', 'https://www.ollama.com', 'https://www.ollama.com/v1', 'https://www.ollama.com:443', 'https://www.ollama.com:443/v1')"
}

// ListOllamaCloudUsageGroupAccounts resolves every sibling for all supplied
// identities with one ID query and one batch hydration. API keys are query
// parameters only; no derived shared key is persisted.
func (r *accountRepository) ListOllamaCloudUsageGroupAccounts(ctx context.Context, accounts []*service.Account) ([]service.Account, error) {
	if r == nil || r.sql == nil {
		return nil, service.ErrOllamaCloudUsageUnavailable
	}
	keys := make([]string, 0, len(accounts))
	seen := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		if !service.IsOllamaCloudUsageAccount(account) || account.Credentials == nil {
			continue
		}
		apiKey, ok := account.Credentials["api_key"].(string)
		if !ok || apiKey == "" {
			continue
		}
		if _, duplicate := seen[apiKey]; duplicate {
			continue
		}
		seen[apiKey] = struct{}{}
		keys = append(keys, apiKey)
	}
	if len(keys) == 0 {
		return []service.Account{}, nil
	}
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = key
	}
	query := `
		SELECT id
		FROM accounts
		WHERE deleted_at IS NULL
		AND ` + ollamaCloudUsageEligibleSQL + `
			AND json_extract(credentials, '$.api_key') IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY id
	`
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := make([]int64, 0, len(keys))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 || r.client == nil {
		return []service.Account{}, nil
	}
	hydrated, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]service.Account, 0, len(hydrated))
	for _, account := range hydrated {
		if account != nil {
			result = append(result, *account)
		}
	}
	return result, nil
}

func (r *accountRepository) SaveOllamaCloudUsageSession(ctx context.Context, account *service.Account, ciphertext string, autoRefresh bool) error {
	return r.updateOllamaCloudUsageGroup(ctx, account, map[string]any{
		service.OllamaCloudUsageSessionExtraKey:     ciphertext,
		service.OllamaCloudUsageAutoRefreshExtraKey: autoRefresh,
	}, false)
}

func (r *accountRepository) DeleteOllamaCloudUsageSession(ctx context.Context, account *service.Account) error {
	return r.updateOllamaCloudUsageGroup(ctx, account, map[string]any{}, false)
}

func (r *accountRepository) SetOllamaCloudUsageAutoRefresh(ctx context.Context, account *service.Account, enabled bool) error {
	if !ollamaCloudUsageAccountHasSession(account) {
		return service.ErrOllamaCloudUsageSessionRequired
	}
	payload := ollamaCloudUsageManagedPayload(account)
	payload[service.OllamaCloudUsageAutoRefreshExtraKey] = enabled
	return r.updateOllamaCloudUsageGroup(ctx, account, payload, true)
}

func (r *accountRepository) UpdateOllamaCloudUsageSnapshot(ctx context.Context, account *service.Account, snapshot *service.OllamaCloudUsageSnapshot) error {
	if account == nil || snapshot == nil {
		return service.ErrAccountNilInput
	}
	if !ollamaCloudUsageAccountHasSession(account) {
		return service.ErrOllamaCloudUsageSessionRequired
	}
	payload := ollamaCloudUsageManagedPayload(account)
	payload[service.OllamaCloudUsageSnapshotExtraKey] = snapshot
	return r.updateOllamaCloudUsageGroup(ctx, account, payload, true)
}

// DisableOllamaCloudUsageAutoRefresh is group-scoped and retains the loaded
// identity CAS. It cannot disable a new group after the account changes key.
func (r *accountRepository) DisableOllamaCloudUsageAutoRefresh(ctx context.Context, account *service.Account) error {
	if !ollamaCloudUsageAccountHasSession(account) {
		return service.ErrOllamaCloudUsageSessionRequired
	}
	payload := ollamaCloudUsageManagedPayload(account)
	payload[service.OllamaCloudUsageAutoRefreshExtraKey] = false
	delete(payload, service.OllamaCloudUsageSnapshotExtraKey)
	return r.updateOllamaCloudUsageGroup(ctx, account, payload, true)
}

func ollamaCloudUsageManagedPayload(account *service.Account) map[string]any {
	payload := make(map[string]any, 3)
	if account == nil || account.Extra == nil {
		return payload
	}
	for _, key := range []string{
		service.OllamaCloudUsageSessionExtraKey,
		service.OllamaCloudUsageAutoRefreshExtraKey,
		service.OllamaCloudUsageSnapshotExtraKey,
	} {
		if value, ok := account.Extra[key]; ok {
			payload[key] = value
		}
	}
	return payload
}

func ollamaCloudUsageAccountHasSession(account *service.Account) bool {
	if account == nil || account.Extra == nil {
		return false
	}
	value, ok := account.Extra[service.OllamaCloudUsageSessionExtraKey].(string)
	return ok && value != ""
}

type lockedOllamaCloudUsageMember struct {
	id            int64
	anchorMatches bool
	sessionJSON   string
	autoJSON      string
	snapshotJSON  string
}

func (r *accountRepository) updateOllamaCloudUsageGroup(
	ctx context.Context,
	account *service.Account,
	payload map[string]any,
	requireExpectedState bool,
) error {
	if account == nil {
		return service.ErrAccountNilInput
	}
	if r == nil || r.client == nil || !service.IsOllamaCloudUsageAccount(account) {
		return service.ErrOllamaCloudUsageUnavailable
	}
	apiKey, ok := account.Credentials["api_key"].(string)
	if !ok || apiKey == "" {
		return service.ErrOllamaCloudUsageAccountInvalid
	}
	apply := func(txCtx context.Context, client *dbent.Client) error {
		matchesProxy, err := lockAndMatchProbeProxyIdentity(txCtx, client, account)
		if err != nil {
			return err
		}
		if !matchesProxy {
			return service.ErrOllamaCloudUsageIdentityChanged
		}
		members, err := lockOllamaCloudUsageGroup(txCtx, client, account, apiKey)
		if err != nil {
			return err
		}
		anchorMatches := false
		for _, member := range members {
			anchorMatches = anchorMatches || member.anchorMatches
		}
		if !anchorMatches {
			return service.ErrOllamaCloudUsageIdentityChanged
		}
		if requireExpectedState {
			expectedSession, err := canonicalAccountExtraJSON(account, service.OllamaCloudUsageSessionExtraKey)
			if err != nil {
				return err
			}
			expectedAuto, err := canonicalAccountExtraJSON(account, service.OllamaCloudUsageAutoRefreshExtraKey)
			if err != nil {
				return err
			}
			expectedSnapshot, err := canonicalAccountExtraJSON(account, service.OllamaCloudUsageSnapshotExtraKey)
			if err != nil {
				return err
			}
			stateMatches := false
			for _, member := range members {
				if canonicalJSON(member.sessionJSON) == expectedSession &&
					canonicalJSON(member.autoJSON) == expectedAuto &&
					canonicalJSON(member.snapshotJSON) == expectedSnapshot {
					stateMatches = true
					break
				}
			}
			if !stateMatches {
				return service.ErrOllamaCloudUsageIdentityChanged
			}
		}
		for _, member := range members {
			current, err := client.Account.Get(txCtx, member.id)
			if err != nil {
				return service.ErrOllamaCloudUsageIdentityChanged
			}
			extra := normalizeJSONMap(current.Extra)
			delete(extra, service.OllamaCloudUsageSessionExtraKey)
			delete(extra, service.OllamaCloudUsageAutoRefreshExtraKey)
			delete(extra, service.OllamaCloudUsageSnapshotExtraKey)
			for key, value := range payload {
				extra[key] = value
			}
			if _, err := client.Account.UpdateOneID(member.id).SetExtra(extra).Save(txCtx); err != nil {
				return err
			}
		}
		return nil
	}
	if dbent.TxFromContext(ctx) != nil {
		return apply(ctx, clientFromContext(ctx, r.client))
	}
	tx, err := r.client.Tx(ctx)
	if errors.Is(err, dbent.ErrTxStarted) {
		return apply(ctx, r.client)
	}
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := apply(txCtx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func lockOllamaCloudUsageGroup(
	ctx context.Context,
	client *dbent.Client,
	account *service.Account,
	apiKey string,
) ([]lockedOllamaCloudUsageMember, error) {
	credentials, err := json.Marshal(normalizeJSONMap(account.Credentials))
	if err != nil {
		return nil, err
	}
	var proxyID any
	if account.ProxyID != nil {
		proxyID = *account.ProxyID
	}
	rows, err := client.QueryContext(ctx, `
		SELECT
			id,
			id = $2
				AND platform = $3
				AND type = $4
				AND json(credentials) = json($5)
				AND proxy_id IS $6,
			COALESCE(extra -> '$.ollama_cloud_usage_session', 'null'),
			COALESCE(extra -> '$.ollama_cloud_usage_auto_refresh', 'null'),
			COALESCE(extra -> '$.ollama_cloud_usage_snapshot', 'null')
		FROM accounts
		WHERE deleted_at IS NULL
			AND `+ollamaCloudUsageEligibleSQL+`
			AND json_extract(credentials, '$.api_key') = $1
		ORDER BY id
	`, apiKey, account.ID, account.Platform, account.Type, string(credentials), proxyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	members := make([]lockedOllamaCloudUsageMember, 0, 1)
	for rows.Next() {
		var member lockedOllamaCloudUsageMember
		if err := rows.Scan(&member.id, &member.anchorMatches, &member.sessionJSON, &member.autoJSON, &member.snapshotJSON); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, service.ErrOllamaCloudUsageIdentityChanged
	}
	return members, nil
}

func canonicalAccountExtraJSON(account *service.Account, key string) (string, error) {
	var value any
	if account != nil && account.Extra != nil {
		value = account.Extra[key]
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return canonicalJSON(string(raw)), nil
}

func canonicalJSON(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// ListDueOllamaCloudUsageAccounts returns at most one truly-due activity-driven
// candidate per exact API key. Due timing (debounce, max-wait, failure backoff)
// is evaluated in SQL before LIMIT so non-due active groups cannot starve due ones.
// Account.LastUsedAt is stamped with the group MAX(last_used_at) for a service
// pure-function recheck against races between list and refresh.
//
// Rules mirror service.ollamaCloudUsageAutoRefreshDueAt (keep both in sync):
//   - missing/invalid snapshot or times → fail-open first due
//   - success: activity after fetched_at;
//     due_at = GREATEST(LEAST(last_used+debounce, fetched+maxWait), fetched+minFetchInterval)
//   - failed/unauthorized: activity after last_attempt; activity_due = LEAST(...);
//     final due_at is not earlier than a valid next_refresh_at (invalid/missing fail-open)
func (r *accountRepository) ListDueOllamaCloudUsageAccounts(
	ctx context.Context,
	now time.Time,
	debounce, maxWait time.Duration,
	limit int,
) ([]service.Account, error) {
	if limit <= 0 {
		return []service.Account{}, nil
	}
	if r == nil || r.sql == nil {
		return nil, errors.New("account repository SQL executor not configured")
	}
	if debounce <= 0 {
		debounce = time.Minute
	}
	if maxWait <= 0 {
		maxWait = time.Hour
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT a.id,
			json_extract(a.credentials, '$.api_key'),
			(SELECT MAX(b.last_used_at)
			 FROM accounts b
			 WHERE b.deleted_at IS NULL
			   AND json_extract(b.credentials, '$.api_key') = json_extract(a.credentials, '$.api_key')),
			json_extract(a.extra, '$.ollama_cloud_usage_snapshot')
		FROM accounts a
		WHERE a.deleted_at IS NULL
			AND a.status = 'active'
			AND a.platform IN ('openai', 'anthropic')
			AND a.type = 'apikey'
			AND lower(trim(json_extract(a.credentials, '$.base_url'))) IN (
				'https://ollama.com', 'https://ollama.com/v1', 'https://ollama.com:443',
				'https://ollama.com:443/v1', 'https://www.ollama.com', 'https://www.ollama.com/v1',
				'https://www.ollama.com:443', 'https://www.ollama.com:443/v1')
			AND json_type(a.credentials, '$.api_key') = 'text'
			AND json_type(a.extra, '$.ollama_cloud_usage_session') = 'text'
			AND json_extract(a.extra, '$.ollama_cloud_usage_auto_refresh') = 1
		ORDER BY a.id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	type dueRow struct {
		id            int64
		apiKey        string
		groupLastUsed *time.Time
		dueClass      int
		dueAt         time.Time
	}
	rowsOut := make([]dueRow, 0, limit)
	for rows.Next() {
		var row dueRow
		var groupLastUsed sql.NullString
		var snapshotJSON sql.NullString
		if err := rows.Scan(&row.id, &row.apiKey, &groupLastUsed, &snapshotJSON); err != nil {
			return nil, err
		}
		if groupLastUsed.Valid {
			if timestamp, ok := parseSQLiteTimestamp(groupLastUsed.String); ok {
				row.groupLastUsed = &timestamp
			}
		}
		row.dueAt, row.dueClass, _ = ollamaCloudUsageRepositoryDueAt(snapshotJSON, row.groupLastUsed, debounce, maxWait)
		if row.dueClass < 0 || now.Before(row.dueAt) {
			continue
		}
		rowsOut = append(rowsOut, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(rowsOut, func(i, j int) bool {
		if rowsOut[i].dueClass != rowsOut[j].dueClass {
			return rowsOut[i].dueClass < rowsOut[j].dueClass
		}
		if !rowsOut[i].dueAt.Equal(rowsOut[j].dueAt) {
			return rowsOut[i].dueAt.Before(rowsOut[j].dueAt)
		}
		return rowsOut[i].id < rowsOut[j].id
	})
	groupSeen := make(map[string]struct{}, len(rowsOut))
	selected := rowsOut[:0]
	for _, row := range rowsOut {
		if _, exists := groupSeen[row.apiKey]; exists {
			continue
		}
		groupSeen[row.apiKey] = struct{}{}
		selected = append(selected, row)
		if len(selected) == limit {
			break
		}
	}
	ids := make([]int64, len(selected))
	for index := range selected {
		ids[index] = selected[index].id
	}
	accounts, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*service.Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}
	result := make([]service.Account, 0, len(rowsOut))
	for _, row := range selected {
		account := byID[row.id]
		if account == nil {
			continue
		}
		// Stamp group MAX(last_used_at) for service due evaluation.
		if row.groupLastUsed != nil {
			ts := row.groupLastUsed.UTC()
			account.LastUsedAt = &ts
		} else {
			account.LastUsedAt = nil
		}
		result = append(result, *account)
	}
	return result, nil
}

func parseSQLiteTimestamp(raw string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999 -0700 MST", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func ollamaCloudUsageRepositoryDueAt(snapshotJSON sql.NullString, groupLastUsed *time.Time, debounce, maxWait time.Duration) (time.Time, int, bool) {
	if !snapshotJSON.Valid || strings.TrimSpace(snapshotJSON.String) == "" || snapshotJSON.String == "null" {
		return time.Time{}, 0, true
	}
	var snapshot service.OllamaCloudUsageSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON.String), &snapshot); err != nil {
		return time.Time{}, 0, true
	}
	switch snapshot.Status {
	case service.OllamaCloudUsageStatusOK:
		if snapshot.FetchedAt == nil || snapshot.FetchedAt.IsZero() {
			return time.Time{}, 0, true
		}
		fetched := snapshot.FetchedAt.UTC()
		if groupLastUsed == nil || !groupLastUsed.After(fetched) {
			return time.Time{}, -1, false
		}
		dueAt := minOllamaCloudUsageTime(groupLastUsed.Add(debounce), fetched.Add(maxWait))
		if floor := fetched.Add(service.OllamaCloudUsageMinFetchInterval); dueAt.Before(floor) {
			dueAt = floor
		}
		return dueAt, 1, true
	case service.OllamaCloudUsageStatusFailed, service.OllamaCloudUsageStatusUnauthorized:
		if snapshot.LastAttemptAt.IsZero() {
			return time.Time{}, 0, true
		}
		attempted := snapshot.LastAttemptAt.UTC()
		if groupLastUsed == nil || !groupLastUsed.After(attempted) {
			return time.Time{}, -1, false
		}
		dueAt := minOllamaCloudUsageTime(groupLastUsed.Add(debounce), attempted.Add(maxWait))
		if !snapshot.NextRefreshAt.IsZero() && snapshot.NextRefreshAt.After(dueAt) {
			dueAt = snapshot.NextRefreshAt.UTC()
		}
		return dueAt, 1, true
	default:
		return time.Time{}, 0, true
	}
}

func minOllamaCloudUsageTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}
