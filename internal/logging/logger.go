package logging

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus for consistent logging across the application.
type Logger struct {
	*logrus.Logger
}

// New creates a new logger instance.
func New() *Logger {
	log := logrus.New()

	// Set default format
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set default output
	log.SetOutput(os.Stdout)

	// Set default level
	log.SetLevel(logrus.InfoLevel)

	return &Logger{log}
}

// NewWithOutput creates a logger with custom output.
func NewWithOutput(out io.Writer) *Logger {
	log := logrus.New()

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	log.SetOutput(out)
	log.SetLevel(logrus.InfoLevel)

	return &Logger{log}
}

// SetLevel sets the logging level.
func (l *Logger) SetLevel(level logrus.Level) {
	l.Logger.SetLevel(level)
}

// WithField returns a new entry with a field.
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields returns a new entry with multiple fields.
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError returns a new entry with an error field.
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}
