package middlewares

import (
	"strings"
	"time"

	"github.com/drama-generator/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldSkipHTTPLog(path) {
			c.Next()
			return
		}

		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		isSlow := duration > time.Second
		isError := status >= 400

		if isError || isSlow {
			log.Warnw("HTTP Request",
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent(),
			)
			return
		}

		// 正常请求只保留精简字段，避免日志过大。
		log.Infow("HTTP Request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

func shouldSkipHTTPLog(path string) bool {
	skipPaths := []string{
		"/health",
		"/favicon.ico",
		"/static/",
	}

	for _, p := range skipPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
