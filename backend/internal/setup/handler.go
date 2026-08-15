package setup

import (
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"

	"github.com/gin-gonic/gin"
)

// installMutex prevents concurrent installation attempts (TOCTOU protection)
var installMutex sync.Mutex

// RegisterRoutes registers setup wizard routes
func RegisterRoutes(r *gin.Engine) {
	setup := r.Group("/setup")
	{
		// Status endpoint is always accessible (read-only)
		setup.GET("/status", getStatus)

		// All modification endpoints are protected by setupGuard
		protected := setup.Group("")
		protected.Use(setupGuard())
		{
			protected.POST("/test-db", testDatabase)
			protected.POST("/test-redis", testRedis)
			protected.POST("/install", install)
		}
	}
}

// SetupStatus represents the current setup state
type SetupStatus struct {
	NeedsSetup bool   `json:"needs_setup"`
	Step       string `json:"step"`
}

// getStatus returns the current setup status
func getStatus(c *gin.Context) {
	response.Success(c, SetupStatus{
		NeedsSetup: NeedsSetup(),
		Step:       "welcome",
	})
}

// setupGuard middleware ensures setup endpoints are only accessible during setup mode
func setupGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !NeedsSetup() {
			response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
			c.Abort()
			return
		}
		c.Next()
	}
}

// validateHostname checks if a hostname/IP is safe (no injection characters)
func validateHostname(host string) bool {
	// Allow only alphanumeric, dots, hyphens, and colons (for IPv6)
	validHost := regexp.MustCompile(`^[a-zA-Z0-9.\-:]+$`)
	return validHost.MatchString(host) && len(host) <= 253
}

// validateEmail checks if email format is valid
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil && len(email) <= 254
}

// validatePassword checks password strength
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters")
	}
	return nil
}

// validatePort checks if port is in valid range
func validatePort(port int) bool {
	return port > 0 && port <= 65535
}


// TestDatabaseRequest represents database test request
type TestDatabaseRequest struct {
	Driver   string `json:"driver"`
	Path     string `json:"path"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// testDatabase tests database connection
func testDatabase(c *gin.Context) {
	var req TestDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	cfg := &DatabaseConfig{
		Driver: "sqlite",
		Path:   req.Path,
	}
	if strings.TrimSpace(cfg.Path) == "" {
		cfg.Path = "./data/sub2api.db"
	}

	if err := TestDatabaseConnection(cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "Connection failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// TestRedisRequest represents Redis test request
type TestRedisRequest struct {
	Enabled   *bool  `json:"enabled"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	DB        int    `json:"db"`
	EnableTLS bool   `json:"enable_tls"`
}

// testRedis tests Redis connection
func testRedis(c *gin.Context) {
	var req TestRedisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	cfg := &RedisConfig{
		Enabled:   req.Enabled,
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Password:  req.Password,
		DB:        req.DB,
		EnableTLS: req.EnableTLS,
	}

	// When Redis is disabled, skip external connectivity checks.
	if !cfg.IsEnabled() {
		response.Success(c, gin.H{"message": "Redis disabled; embedded Redis will be used at runtime"})
		return
	}

	// Security: Validate inputs
	if !validateHostname(req.Host) {
		response.Error(c, http.StatusBadRequest, "Invalid hostname format")
		return
	}
	if !validatePort(req.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid port number")
		return
	}
	if req.DB < 0 || req.DB > 15 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis database number (0-15)")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) > 128 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis username")
		return
	}
	cfg.Username = req.Username

	if err := TestRedisConnection(cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "Connection failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// InstallRequest represents installation request
type InstallRequest struct {
	Database DatabaseConfig `json:"database" binding:"required"`
	Redis    RedisConfig    `json:"redis" binding:"required"`
	Admin    AdminConfig    `json:"admin" binding:"required"`
	Server   ServerConfig   `json:"server"`
}

// install performs the installation
func install(c *gin.Context) {
	// TOCTOU Protection: Acquire mutex to prevent concurrent installation
	installMutex.Lock()
	defer installMutex.Unlock()

	// Double-check after acquiring lock
	if !NeedsSetup() {
		response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
		return
	}

	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	req.Admin.Email = strings.TrimSpace(req.Admin.Email)
	req.Database.Path = strings.TrimSpace(req.Database.Path)
	req.Redis.Host = strings.TrimSpace(req.Redis.Host)
	req.Redis.Username = strings.TrimSpace(req.Redis.Username)

	// ========== COMPREHENSIVE INPUT VALIDATION ==========
	// SQLite is the only supported database target.
	req.Database.Driver = "sqlite"
	if req.Database.Path == "" {
		req.Database.Path = "./data/sub2api.db"
	}

	// Redis validation (optional when disabled)
	if req.Redis.IsEnabled() {
		if !validateHostname(req.Redis.Host) {
			response.Error(c, http.StatusBadRequest, "Invalid Redis hostname")
			return
		}
		if !validatePort(req.Redis.Port) {
			response.Error(c, http.StatusBadRequest, "Invalid Redis port")
			return
		}
		if req.Redis.DB < 0 || req.Redis.DB > 15 {
			response.Error(c, http.StatusBadRequest, "Invalid Redis database number")
			return
		}
		if len(req.Redis.Username) > 128 {
			response.Error(c, http.StatusBadRequest, "Invalid Redis username")
			return
		}
	}

	// Admin validation
	if !validateEmail(req.Admin.Email) {
		response.Error(c, http.StatusBadRequest, "Invalid admin email format")
		return
	}
	if err := validatePassword(req.Admin.Password); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Server validation
	if req.Server.Port != 0 && !validatePort(req.Server.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid server port")
		return
	}

	// ========== SET DEFAULTS ==========
	if req.Server.Host == "" {
		req.Server.Host = "0.0.0.0"
	}
	if req.Server.Port == 0 {
		req.Server.Port = 8080
	}
	if req.Server.Mode == "" {
		req.Server.Mode = "release"
	}
	// Validate server mode
	if req.Server.Mode != "release" && req.Server.Mode != "debug" {
		response.Error(c, http.StatusBadRequest, "Invalid server mode (must be 'release' or 'debug')")
		return
	}

	cfg := &SetupConfig{
		Database: req.Database,
		Redis:    req.Redis,
		Admin:    req.Admin,
		Server:   req.Server,
		JWT: JWTConfig{
			ExpireHour: 24,
		},
	}

	if err := Install(cfg); err != nil {
		response.Error(c, http.StatusInternalServerError, "Installation failed: "+err.Error())
		return
	}

	// Schedule service restart in background after sending response
	// This ensures the client receives the success response before the service restarts
	go func() {
		// Wait a moment to ensure the response is sent
		time.Sleep(500 * time.Millisecond)
		sysutil.RestartServiceAsync()
	}()

	response.Success(c, gin.H{
		"message": "Installation completed successfully. Service will restart automatically.",
		"restart": true,
	})
}
