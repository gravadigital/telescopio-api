package logger

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

// Init initializes the logger with default settings
func Init() {
	Initialize("info")
}

// Initialize sets up the global logger with Charm's log library
func Initialize(logLevel string) {
	Logger = log.New(os.Stderr)

	level := strings.ToLower(logLevel)
	switch level {
	case "debug":
		Logger.SetLevel(log.DebugLevel)
	case "info":
		Logger.SetLevel(log.InfoLevel)
	case "warn", "warning":
		Logger.SetLevel(log.WarnLevel)
	case "error":
		Logger.SetLevel(log.ErrorLevel)
	case "fatal":
		Logger.SetLevel(log.FatalLevel)
	default:
		Logger.SetLevel(log.InfoLevel)
	}

	Logger.SetReportCaller(true)
	Logger.SetReportTimestamp(true)

	Logger.Debug("Logger initialized", "level", level)
}

// Get returns the global logger instance
func Get() *log.Logger {
	if Logger == nil {
		Initialize("info")
	}
	return Logger
}

// WithContext creates a new logger with additional context fields
func WithContext(fields ...any) *log.Logger {
	return Get().With(fields...)
}

// Service creates a logger for a specific service
func Service(serviceName string) *log.Logger {
	return WithContext("service", serviceName)
}

// Database creates a logger for database operations
func Database() *log.Logger {
	return WithContext("component", "database")
}

// HTTP creates a logger for HTTP operations
func HTTP() *log.Logger {
	return WithContext("component", "http")
}

// Migration creates a logger for migration operations
func Migration() *log.Logger {
	return WithContext("component", "migration")
}

// Mathematical creates a logger for mathematical operations
func Mathematical() *log.Logger {
	return WithContext("component", "mathematical")
}

// Repository creates a logger for repository operations
func Repository(repoName string) *log.Logger {
	return WithContext("component", "repository", "repository", repoName)
}

// Handler creates a logger for HTTP handlers
func Handler(handlerName string) *log.Logger {
	return WithContext("component", "handler", "handler", handlerName)
}
