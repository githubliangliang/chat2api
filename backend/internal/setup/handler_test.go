package setup

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInstallNotifiesCompletionAfterSuccess(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SKIP_SETUP", "false")

	recorder := httptest.NewRecorder()
	notifications := 0
	router := gin.New()
	RegisterRoutes(router, func() {
		if recorder.Code != http.StatusOK {
			t.Errorf("completion notified before success response was written: status=%d", recorder.Code)
		}
		notifications++
	})

	body := fmt.Sprintf(`{
		"database":{"driver":"sqlite","path":%q},
		"redis":{"enabled":false,"host":"","port":6379,"username":"","password":"","db":0,"enable_tls":false},
		"admin":{"email":"admin@example.com","password":"password123"},
		"server":{"host":"0.0.0.0","port":8080,"mode":"release"}
	}`, filepath.Join(dataDir, "sub2api.db"))
	request := httptest.NewRequest(http.MethodPost, "/setup/install", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if notifications != 1 {
		t.Fatalf("completion notifications = %d, want 1", notifications)
	}
}

func TestInstallDoesNotNotifyCompletionAfterValidationFailure(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("SKIP_SETUP", "false")

	notifications := 0
	router := gin.New()
	RegisterRoutes(router, func() {
		notifications++
	})

	body := `{
		"database":{"driver":"sqlite","path":"./data/sub2api.db"},
		"redis":{"enabled":false,"host":"","port":6379,"username":"","password":"","db":0,"enable_tls":false},
		"admin":{"email":"admin@example.com","password":"short"},
		"server":{"host":"0.0.0.0","port":8080,"mode":"release"}
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/setup/install", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if notifications != 0 {
		t.Fatalf("completion notifications = %d, want 0", notifications)
	}
}
