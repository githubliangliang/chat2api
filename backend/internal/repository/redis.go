package repository

import (
	"crypto/tls"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

var (
	embeddedRedisOnce sync.Once
	embeddedRedis     *miniredis.Miniredis
	embeddedRedisErr  error
)

// InitRedis 初始化 Redis 客户端。
//
// Redis 为可选组件：
//   - redis.enabled=true（默认）：连接外部 Redis
//   - redis.enabled=false：启动进程内嵌入式 Redis（miniredis），适合单机部署
//
// 注意：嵌入式模式不支持多实例共享状态（分布式锁/缓存跨节点不生效）。
//
// 性能优化说明：
// 1. PoolSize: 控制最大并发连接数（默认 128）
// 2. MinIdleConns: 保持最小空闲连接，减少冷启动延迟（默认 10）
// 3. DialTimeout/ReadTimeout/WriteTimeout: 精确控制各阶段超时
func InitRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(buildRedisOptions(cfg))
	if cfg.Server.EnableServerTiming {
		client.AddHook(serverTimingRedisHook{})
	}
	return client
}

// buildRedisOptions 构建 Redis 连接选项
// 从配置文件读取连接池和超时参数，支持生产环境调优
func buildRedisOptions(cfg *config.Config) *redis.Options {
	addr := cfg.Redis.Address()
	username := cfg.Redis.Username
	password := cfg.Redis.Password
	db := cfg.Redis.DB
	enableTLS := cfg.Redis.EnableTLS

	if !cfg.Redis.IsEnabled() {
		// Single-node fallback: embedded in-process Redis.
		mr, err := getEmbeddedRedis()
		if err != nil {
			// Fall through with original address; connection will fail loudly at first use.
			logger.LegacyPrintf("repository", "failed to start embedded redis: %v", err)
		} else {
			addr = mr.Addr()
			username = ""
			password = ""
			db = 0
			enableTLS = false
			logger.LegacyPrintf("repository", "redis disabled: using embedded in-process redis at %s", addr)
		}
	}

	opts := &redis.Options{
		Addr:         addr,
		Username:     username,
		Password:     password,
		DB:           db,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeoutSeconds) * time.Second,  // 建连超时
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeoutSeconds) * time.Second,  // 读取超时
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeoutSeconds) * time.Second, // 写入超时
		PoolSize:     cfg.Redis.PoolSize,                                         // 连接池大小
		MinIdleConns: cfg.Redis.MinIdleConns,                                     // 最小空闲连接
	}

	if enableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Redis.Host,
		}
	}

	return opts
}

// getEmbeddedRedis starts a process-local miniredis once and keeps it alive for the process lifetime.
func getEmbeddedRedis() (*miniredis.Miniredis, error) {
	embeddedRedisOnce.Do(func() {
		embeddedRedis, embeddedRedisErr = miniredis.Run()
	})
	return embeddedRedis, embeddedRedisErr
}

// CloseEmbeddedRedis shuts down the embedded Redis if it was started.
// Safe to call multiple times; primarily used in tests/cleanup hooks.
func CloseEmbeddedRedis() {
	if embeddedRedis != nil {
		embeddedRedis.Close()
	}
}
