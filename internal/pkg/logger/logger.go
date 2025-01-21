package logger

import (
	"log"
)

// LogLevel type for defining levels of logging
type LogLevel int

const (
	InfoLevel LogLevel = iota
	WarningLevel
	ErrorLevel
)

// Logger struct defines the logger configuration
type Logger struct {
	level LogLevel
}

// NewLogger creates a new Logger instance with the specified log level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level: level,
	}
}

// Info logs a message at InfoLevel. It will print anything that is InfoLevel and above.
func (l *Logger) Info(msg string) {
	if l.level <= InfoLevel {
		log.Println("INFO: " + msg)
	}
}

// Warning logs a message at WarningLevel. It will print anything that is WarningLevel and above.
func (l *Logger) Warning(msg string) {
	if l.level <= WarningLevel {
		log.Println("WARNING: " + msg)
	}
}

// Error logs a message at ErrorLevel.
func (l *Logger) Error(msg string) {
	if l.level <= ErrorLevel {
		log.Println("ERROR: " + msg)
	}
}
