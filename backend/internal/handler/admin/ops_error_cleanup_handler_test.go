package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type opsErrorCleanupRepo struct {
	service.OpsRepository
	called bool
}

func (r *opsErrorCleanupRepo) DeleteAllErrorLogs(context.Context) (int64, error) {
	r.called = true
	return 7, nil
}

func newOpsErrorCleanupRouter(handler *OpsHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withUser {
		r.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 99})
			c.Next()
		})
	}
	r.POST("/errors/cleanup", handler.DeleteAllErrorLogs)
	return r
}

func TestOpsErrorCleanupHandlerDeletesAllLogs(t *testing.T) {
	repo := &opsErrorCleanupRepo{}
	svc := service.NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	r := newOpsErrorCleanupRouter(NewOpsHandler(svc), true)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/errors/cleanup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", w.Code, w.Body.String())
	}
	if !repo.called {
		t.Fatal("DeleteAllErrorLogs was not called")
	}
	var resp responseEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var data struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil || data.Deleted != 7 {
		t.Fatalf("response data=%s err=%v, want deleted=7", resp.Data, err)
	}
}

func TestOpsErrorCleanupHandlerRequiresUser(t *testing.T) {
	svc := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	r := newOpsErrorCleanupRouter(NewOpsHandler(svc), false)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/errors/cleanup", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}
