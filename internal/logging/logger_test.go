package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	logger := New()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	assert.Equal(t, logrus.InfoLevel, logger.Logger.Level)
}

func TestNewWithOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	assert.Equal(t, logrus.InfoLevel, logger.Logger.Level)

	logger.Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Contains(t, logEntry, "timestamp")
}

func TestSetLevel(t *testing.T) {
	logger := New()

	logger.SetLevel(logrus.DebugLevel)
	assert.Equal(t, logrus.DebugLevel, logger.Logger.Level)

	logger.SetLevel(logrus.WarnLevel)
	assert.Equal(t, logrus.WarnLevel, logger.Logger.Level)

	logger.SetLevel(logrus.ErrorLevel)
	assert.Equal(t, logrus.ErrorLevel, logger.Logger.Level)
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	entry := logger.WithField("key", "value")
	assert.NotNil(t, entry)

	entry.Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "value", logEntry["key"])
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	fields := logrus.Fields{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	entry := logger.WithFields(fields)
	assert.NotNil(t, entry)

	entry.Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, float64(123), logEntry["key2"])
	assert.Equal(t, true, logEntry["key3"])
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	testErr := errors.New("test error")
	entry := logger.WithError(testErr)
	assert.NotNil(t, entry)

	entry.Error("error occurred")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "error occurred", logEntry["message"])
	assert.Equal(t, "error", logEntry["level"])
	assert.Equal(t, "test error", logEntry["error"])
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel logrus.Level
		logFunc  func(*Logger, string)
		expected string
	}{
		{
			name:     "Debug",
			logLevel: logrus.DebugLevel,
			logFunc:  func(l *Logger, msg string) { l.Debug(msg) },
			expected: "debug",
		},
		{
			name:     "Info",
			logLevel: logrus.InfoLevel,
			logFunc:  func(l *Logger, msg string) { l.Info(msg) },
			expected: "info",
		},
		{
			name:     "Warn",
			logLevel: logrus.WarnLevel,
			logFunc:  func(l *Logger, msg string) { l.Warn(msg) },
			expected: "warning",
		},
		{
			name:     "Error",
			logLevel: logrus.ErrorLevel,
			logFunc:  func(l *Logger, msg string) { l.Error(msg) },
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewWithOutput(&buf)
			logger.SetLevel(tt.logLevel)

			tt.logFunc(logger, "test message")

			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)

			assert.Equal(t, "test message", logEntry["message"])
			assert.Equal(t, tt.expected, logEntry["level"])
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	logger.Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Contains(t, logEntry, "timestamp")
	assert.Contains(t, logEntry, "level")
	assert.Contains(t, logEntry, "message")
}

func TestLoggerChaining(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	logger.WithField("key1", "value1").
		WithField("key2", "value2").
		WithError(errors.New("test error")).
		Info("chained message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "chained message", logEntry["message"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "test error", logEntry["error"])
}

func TestLoggerWithMultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)

	logger.Info("message 1")
	logger.Info("message 2")
	logger.Info("message 3")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 3)

	for i, line := range lines {
		var logEntry map[string]interface{}
		err := json.Unmarshal(line, &logEntry)
		require.NoError(t, err)
		assert.Contains(t, logEntry["message"], "message")
		_ = i
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 2)

	var logEntry1 map[string]interface{}
	err := json.Unmarshal(lines[0], &logEntry1)
	require.NoError(t, err)
	assert.Equal(t, "warn message", logEntry1["message"])

	var logEntry2 map[string]interface{}
	err = json.Unmarshal(lines[1], &logEntry2)
	require.NoError(t, err)
	assert.Equal(t, "error message", logEntry2["message"])
}
