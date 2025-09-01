// Package events provides middleware for event handling and logging
package events

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

// CreateEvent returns a middleware function that logs request details
func CreateEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		requestID := generateRequestID()
		c.Set("request_id", requestID)

		log.Info("Request started",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"remote_addr", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)

		c.Next()

		latency := time.Since(startTime)
		status := c.Writer.Status()

		logLevel := log.Info
		if status >= 400 {
			logLevel = log.Error
		} else if status >= 300 {
			logLevel = log.Warn
		}

		logLevel("Request completed",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency", latency,
			"size", c.Writer.Size(),
		)
	}
}

// generateRequestID creates a simple request ID for tracing
func generateRequestID() string {
	return "req_" + time.Now().Format("20060102150405") + "_" + time.Now().Format("000000")
}
