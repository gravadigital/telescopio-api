// Package events
package events

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func CreateEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		// Set example variable
		c.Set("example", "12345")

		// before request
		c.Next()

		// after request
		latency := time.Since(t)
		log.Debug(latency)

		// access the status we are sending
		status := c.Writer.Status()
		log.Debug(status)
	}
}
