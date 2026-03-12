package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"rankmyapp/pkg/logger"
)

func RequestLogger() gin.HandlerFunc {
	log := logger.Get()
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			log.Error("request", append(fields, zap.String("errors", c.Errors.String()))...)
		} else {
			log.Info("request", fields...)
		}
	}
}
