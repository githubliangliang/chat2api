package service

import (
	"context"
	"errors"
	"testing"
)

func TestOpsServiceDeleteAllErrorLogs(t *testing.T) {
	called := false
	repo := &opsRepoMock{DeleteAllErrorLogsFn: func(context.Context) (int64, error) {
		called = true
		return 7, nil
	}}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	deleted, err := svc.DeleteAllErrorLogs(context.Background())
	if err != nil {
		t.Fatalf("DeleteAllErrorLogs() error: %v", err)
	}
	if !called || deleted != 7 {
		t.Fatalf("DeleteAllErrorLogs() called=%v deleted=%d, want called=true deleted=7", called, deleted)
	}
}

func TestOpsServiceDeleteAllErrorLogsPropagatesError(t *testing.T) {
	wantErr := errors.New("delete failed")
	repo := &opsRepoMock{DeleteAllErrorLogsFn: func(context.Context) (int64, error) { return 0, wantErr }}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	if _, err := svc.DeleteAllErrorLogs(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("DeleteAllErrorLogs() error = %v, want %v", err, wantErr)
	}
}
