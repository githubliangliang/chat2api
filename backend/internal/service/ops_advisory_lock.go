package service

import (
	"context"
	"database/sql"
	"hash/fnv"
	"sync"
)

func hashAdvisoryLockID(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return int64(h.Sum64())
}

var dbLeaderLocks = newProcessLeaderLockRegistry()

type processLeaderLockRegistry struct {
	mu   sync.Mutex
	held map[int64]struct{}
}

func newProcessLeaderLockRegistry() *processLeaderLockRegistry {
	return &processLeaderLockRegistry{held: make(map[int64]struct{})}
}

func (r *processLeaderLockRegistry) tryAcquire(lockID int64) (func(), bool) {
	r.mu.Lock()
	if _, exists := r.held[lockID]; exists {
		r.mu.Unlock()
		return nil, false
	}
	r.held[lockID] = struct{}{}
	r.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			delete(r.held, lockID)
			r.mu.Unlock()
		})
	}, true
}

func tryAcquireDBAdvisoryLock(ctx context.Context, db *sql.DB, lockID int64) (func(), bool) {
	release, acquired, _ := tryAcquireDBAdvisoryLockWithError(ctx, db, lockID)
	return release, acquired
}

func tryAcquireDBAdvisoryLockWithError(_ context.Context, db *sql.DB, lockID int64) (func(), bool, error) {
	if db == nil {
		return nil, false, nil
	}
	release, acquired := dbLeaderLocks.tryAcquire(lockID)
	return release, acquired, nil
}
