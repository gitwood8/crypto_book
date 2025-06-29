package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	infoLogger  = log.New(os.Stdout, "", 0)
	errorLogger = log.New(os.Stderr, "", 0)
	debugLogger = log.New(os.Stdout, "", 0)
	warnLogger  = log.New(os.Stdout, "", 0)
)

// formatKeyValues formats key-value pairs into a readable string
func formatKeyValues(kvs ...any) string {
	if len(kvs) == 0 {
		return ""
	}

	var parts []string
	for i := 0; i < len(kvs); i += 2 {
		if i+1 < len(kvs) {
			key := fmt.Sprint(kvs[i])
			value := fmt.Sprint(kvs[i+1])
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		} else {
			// Odd number of arguments, treat last one as a standalone value
			parts = append(parts, fmt.Sprint(kvs[i]))
		}
	}
	return strings.Join(parts, " ")
}

// logWithCaller logs a message with proper file name and line number
func logWithCaller(logger *log.Logger, level string, format string, v ...any) {
	// Get caller information (skip 2 frames: logWithCaller and the wrapper function)
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Get just the filename, not the full path
	filename := filepath.Base(file)

	// Create timestamp
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	// Create the log message with proper format
	var message string
	if format == "" {
		if len(v) == 0 {
			message = ""
		} else if len(v) == 1 {
			// Single message
			message = fmt.Sprint(v[0])
		} else {
			// Message + key-value pairs
			message = fmt.Sprint(v[0])
			kvs := formatKeyValues(v[1:]...)
			if kvs != "" {
				message += " " + kvs
			}
		}
	} else {
		// For formatted messages (Infof, Errorf, etc.)
		message = fmt.Sprintf(format, v...)
	}

	// Log with proper format: timestamp level filename:line message
	logger.Printf("%s %s %s:%d %s", timestamp, level, filename, line, message)
}

// Info logs an informational message with optional key-value pairs
// Usage: log.Info("message") or log.Info("message", "key", value, "key2", value2)
func Info(v ...any) {
	logWithCaller(infoLogger, "INFO:", "", v...)
}

func Infof(format string, v ...any) {
	logWithCaller(infoLogger, "INFO:", format, v...)
}

func Error(v ...any) {
	logWithCaller(errorLogger, "ERROR:", "", v...)
}

func Errorf(format string, v ...any) {
	logWithCaller(errorLogger, "ERROR:", format, v...)
}

func Warn(v ...any) {
	logWithCaller(warnLogger, "WARN:", "", v...)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...any) {
	logWithCaller(warnLogger, "WARN:", format, v...)
}

func Debug(v ...any) {
	logWithCaller(debugLogger, "DEBUG:", "", v...)
}

func Debugf(format string, v ...any) {
	logWithCaller(debugLogger, "DEBUG:", format, v...)
}
