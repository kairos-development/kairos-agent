package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogViewer(t *testing.T) {
	lv := NewLogViewer(100)
	assert.NotNil(t, lv)
	assert.Equal(t, 100, lv.maxEntries)
	assert.True(t, lv.showTimestamp)
	assert.Empty(t, lv.entries)
}

func TestLogViewer_AddLog(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Test message")

	assert.Len(t, lv.entries, 1)
	assert.Equal(t, LogLevelInfo, lv.entries[0].Level)
	assert.Equal(t, "Test message", lv.entries[0].Message)
}

func TestLogViewer_AddLog_TrimOldEntries(t *testing.T) {
	lv := NewLogViewer(5)

	// Add more than max entries
	for i := 0; i < 10; i++ {
		lv.AddLog(LogLevelInfo, "Message")
	}

	assert.Len(t, lv.entries, 5)
}

func TestLogViewer_SetSize(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 20)

	assert.Equal(t, 80, lv.width)
	assert.Equal(t, 20, lv.height)
}

func TestLogViewer_ScrollUp(t *testing.T) {
	lv := NewLogViewer(100)
	lv.offset = 5

	lv.ScrollUp()
	assert.Equal(t, 4, lv.offset)

	// Can't scroll below 0
	lv.offset = 0
	lv.ScrollUp()
	assert.Equal(t, 0, lv.offset)
}

func TestLogViewer_ScrollDown(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 5)

	// Add 10 entries
	for i := 0; i < 10; i++ {
		lv.AddLog(LogLevelInfo, "Message")
	}

	lv.offset = 0
	lv.ScrollDown()
	assert.Equal(t, 1, lv.offset)
}

func TestLogViewer_ScrollToBottom(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 5)

	// Add 10 entries
	for i := 0; i < 10; i++ {
		lv.AddLog(LogLevelInfo, "Message")
	}

	lv.offset = 0
	lv.ScrollToBottom()
	assert.Equal(t, 5, lv.offset) // 10 entries - 5 height = 5
}

func TestLogViewer_SetFilterLevel(t *testing.T) {
	lv := NewLogViewer(100)
	lv.offset = 5

	level := LogLevelError
	lv.SetFilterLevel(&level)

	assert.Equal(t, &level, lv.filterLevel)
	assert.Equal(t, 0, lv.offset) // Reset offset
}

func TestLogViewer_SetFilterText(t *testing.T) {
	lv := NewLogViewer(100)
	lv.offset = 5

	lv.SetFilterText("error")

	assert.Equal(t, "error", lv.filterText)
	assert.Equal(t, 0, lv.offset) // Reset offset
}

func TestLogViewer_ToggleTimestamp(t *testing.T) {
	lv := NewLogViewer(100)
	assert.True(t, lv.showTimestamp)

	lv.ToggleTimestamp()
	assert.False(t, lv.showTimestamp)

	lv.ToggleTimestamp()
	assert.True(t, lv.showTimestamp)
}

func TestLogViewer_ClearFilters(t *testing.T) {
	lv := NewLogViewer(100)
	level := LogLevelError
	lv.SetFilterLevel(&level)
	lv.SetFilterText("error")
	lv.offset = 5

	lv.ClearFilters()

	assert.Nil(t, lv.filterLevel)
	assert.Empty(t, lv.filterText)
	assert.Equal(t, 0, lv.offset)
}

func TestLogViewer_GetFilteredEntries_NoFilter(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Info message")
	lv.AddLog(LogLevelError, "Error message")

	filtered := lv.getFilteredEntries()
	assert.Len(t, filtered, 2)
}

func TestLogViewer_GetFilteredEntries_LevelFilter(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Info message")
	lv.AddLog(LogLevelError, "Error message")
	lv.AddLog(LogLevelWarning, "Warning message")

	level := LogLevelError
	lv.SetFilterLevel(&level)

	filtered := lv.getFilteredEntries()
	assert.Len(t, filtered, 1)
	assert.Equal(t, LogLevelError, filtered[0].Level)
}

func TestLogViewer_GetFilteredEntries_TextFilter(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Connection established")
	lv.AddLog(LogLevelError, "Connection failed")
	lv.AddLog(LogLevelInfo, "Order submitted")

	lv.SetFilterText("connection")

	filtered := lv.getFilteredEntries()
	assert.Len(t, filtered, 2)
}

func TestLogViewer_GetFilteredEntries_CaseInsensitive(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Connection Established")

	lv.SetFilterText("CONNECTION")

	filtered := lv.getFilteredEntries()
	assert.Len(t, filtered, 1)
}

func TestLogViewer_GetFilteredEntries_BothFilters(t *testing.T) {
	lv := NewLogViewer(100)
	lv.AddLog(LogLevelInfo, "Connection established")
	lv.AddLog(LogLevelError, "Connection failed")
	lv.AddLog(LogLevelError, "Order rejected")

	level := LogLevelError
	lv.SetFilterLevel(&level)
	lv.SetFilterText("connection")

	filtered := lv.getFilteredEntries()
	assert.Len(t, filtered, 1)
	assert.Equal(t, "Connection failed", filtered[0].Message)
}

func TestLogViewer_Render_Empty(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)

	result := lv.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "No logs yet")
}

func TestLogViewer_Render_EmptyWithFilters(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)
	lv.AddLog(LogLevelInfo, "Info message")

	level := LogLevelError
	lv.SetFilterLevel(&level)

	result := lv.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "No logs match current filters")
}

func TestLogViewer_Render_WithEntries(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)
	lv.AddLog(LogLevelInfo, "Test message")

	result := lv.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test message")
}

func TestLogViewer_Render_AllLogLevels(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)

	lv.AddLog(LogLevelInfo, "Info message")
	lv.AddLog(LogLevelWarning, "Warning message")
	lv.AddLog(LogLevelError, "Error message")
	lv.AddLog(LogLevelSuccess, "Success message")

	result := lv.Render()
	assert.NotEmpty(t, result)
}

func TestLogViewer_RenderEntry_WithTimestamp(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)
	lv.showTimestamp = true

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "Test message",
	}

	result := lv.renderEntry(entry)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test message")
}

func TestLogViewer_RenderEntry_WithoutTimestamp(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)
	lv.showTimestamp = false

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "Test message",
	}

	result := lv.renderEntry(entry)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test message")
}

func TestLogViewer_RenderEntry_LongMessage(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetSize(80, 10)

	longMessage := "This is a very long message that should be truncated because it exceeds the maximum width of the log viewer component"
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   longMessage,
	}

	result := lv.renderEntry(entry)
	assert.NotEmpty(t, result)
	// Should be truncated
	assert.Less(t, len(result), len(longMessage))
}

func TestLogViewer_RenderFilterInfo_NoFilters(t *testing.T) {
	lv := NewLogViewer(100)

	result := lv.renderFilterInfo()
	assert.Empty(t, result)
}

func TestLogViewer_RenderFilterInfo_LevelFilter(t *testing.T) {
	lv := NewLogViewer(100)
	level := LogLevelError
	lv.SetFilterLevel(&level)

	result := lv.renderFilterInfo()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "ERROR")
}

func TestLogViewer_RenderFilterInfo_TextFilter(t *testing.T) {
	lv := NewLogViewer(100)
	lv.SetFilterText("error")

	result := lv.renderFilterInfo()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "error")
}

func TestLogViewer_RenderFilterInfo_BothFilters(t *testing.T) {
	lv := NewLogViewer(100)
	level := LogLevelWarning
	lv.SetFilterLevel(&level)
	lv.SetFilterText("connection")

	result := lv.renderFilterInfo()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "WARN")
	assert.Contains(t, result, "connection")
}

func TestLogViewer_RenderFilterInfo_AllLevels(t *testing.T) {
	lv := NewLogViewer(100)

	levels := []LogLevel{LogLevelInfo, LogLevelWarning, LogLevelError, LogLevelSuccess}
	expected := []string{"INFO", "WARN", "ERROR", "OK"}

	for i, level := range levels {
		l := level
		lv.SetFilterLevel(&l)
		result := lv.renderFilterInfo()
		assert.Contains(t, result, expected[i])
	}
}
