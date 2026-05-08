package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
)

// LogLevel defines the severity of a log entry.
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarning
	LogLevelError
	LogLevelSuccess
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

// LogViewer displays scrollable logs with filtering.
type LogViewer struct {
	entries       []LogEntry
	maxEntries    int
	height        int
	width         int
	offset        int
	filterLevel   *LogLevel // nil = show all
	filterText    string
	showTimestamp bool
}

// NewLogViewer creates a new log viewer.
func NewLogViewer(maxEntries int) *LogViewer {
	return &LogViewer{
		entries:       make([]LogEntry, 0, maxEntries),
		maxEntries:    maxEntries,
		showTimestamp: true,
	}
}

// AddLog adds a new log entry.
func (lv *LogViewer) AddLog(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
	}

	lv.entries = append(lv.entries, entry)

	// Trim old entries
	if len(lv.entries) > lv.maxEntries {
		lv.entries = lv.entries[len(lv.entries)-lv.maxEntries:]
	}

	// Auto-scroll to bottom
	lv.ScrollToBottom()
}

// SetSize sets the dimensions of the log viewer.
func (lv *LogViewer) SetSize(width, height int) {
	lv.width = width
	lv.height = height
}

// ScrollUp scrolls the log viewer up.
func (lv *LogViewer) ScrollUp() {
	if lv.offset > 0 {
		lv.offset--
	}
}

// ScrollDown scrolls the log viewer down.
func (lv *LogViewer) ScrollDown() {
	filtered := lv.getFilteredEntries()
	maxOffset := len(filtered) - lv.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if lv.offset < maxOffset {
		lv.offset++
	}
}

// ScrollToBottom scrolls to the bottom of the logs.
func (lv *LogViewer) ScrollToBottom() {
	filtered := lv.getFilteredEntries()
	maxOffset := len(filtered) - lv.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	lv.offset = maxOffset
}

// SetFilterLevel sets the log level filter (nil = show all).
func (lv *LogViewer) SetFilterLevel(level *LogLevel) {
	lv.filterLevel = level
	lv.offset = 0
}

// SetFilterText sets the text filter (empty = show all).
func (lv *LogViewer) SetFilterText(text string) {
	lv.filterText = text
	lv.offset = 0
}

// ToggleTimestamp toggles timestamp display.
func (lv *LogViewer) ToggleTimestamp() {
	lv.showTimestamp = !lv.showTimestamp
}

// ClearFilters clears all filters.
func (lv *LogViewer) ClearFilters() {
	lv.filterLevel = nil
	lv.filterText = ""
	lv.offset = 0
}

// getFilteredEntries returns entries matching current filters.
func (lv *LogViewer) getFilteredEntries() []LogEntry {
	if lv.filterLevel == nil && lv.filterText == "" {
		return lv.entries
	}

	filtered := make([]LogEntry, 0, len(lv.entries))
	for _, entry := range lv.entries {
		// Level filter
		if lv.filterLevel != nil && entry.Level != *lv.filterLevel {
			continue
		}

		// Text filter
		if lv.filterText != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(lv.filterText)) {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// Render renders the log viewer.
func (lv *LogViewer) Render() string {
	filtered := lv.getFilteredEntries()

	if len(filtered) == 0 {
		emptyMsg := "No logs yet..."
		if lv.filterLevel != nil || lv.filterText != "" {
			emptyMsg = "No logs match current filters"
		}
		return styles.BorderStyle.
			Width(lv.width - 4).
			Height(lv.height - 2).
			Render(styles.HelpStyle.Render(emptyMsg))
	}

	var lines []string

	// Calculate visible range
	start := lv.offset
	end := lv.offset + lv.height
	if end > len(filtered) {
		end = len(filtered)
	}

	// Render visible entries
	for i := start; i < end; i++ {
		entry := filtered[i]
		lines = append(lines, lv.renderEntry(entry))
	}

	// Fill remaining space
	for len(lines) < lv.height {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Add filter indicator
	filterInfo := lv.renderFilterInfo()
	if filterInfo != "" {
		content = filterInfo + "\n" + content
	}

	return styles.BorderStyle.
		Width(lv.width - 4).
		Height(lv.height - 2).
		Render(content)
}

// renderEntry renders a single log entry.
func (lv *LogViewer) renderEntry(entry LogEntry) string {
	var parts []string

	// Timestamp (optional)
	if lv.showTimestamp {
		timestamp := entry.Timestamp.Format("15:04:05")
		timestampStr := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(timestamp)
		parts = append(parts, timestampStr)
	}

	var levelStyle lipgloss.Style
	var levelStr string

	switch entry.Level {
	case LogLevelInfo:
		levelStyle = styles.LogInfoStyle
		levelStr = "INFO "
	case LogLevelWarning:
		levelStyle = styles.LogWarningStyle
		levelStr = "WARN "
	case LogLevelError:
		levelStyle = styles.LogErrorStyle
		levelStr = "ERROR"
	case LogLevelSuccess:
		levelStyle = styles.LogSuccessStyle
		levelStr = "OK   "
	}

	level := levelStyle.Render(levelStr)
	message := levelStyle.Render(entry.Message)

	parts = append(parts, level, message)

	// Truncate if too long
	result := strings.Join(parts, " ")
	maxWidth := lv.width - 8
	if lipgloss.Width(result) > maxWidth {
		result = result[:maxWidth-3] + "..."
	}

	return result
}

// renderFilterInfo renders the filter status line.
func (lv *LogViewer) renderFilterInfo() string {
	if lv.filterLevel == nil && lv.filterText == "" {
		return ""
	}

	var filters []string

	if lv.filterLevel != nil {
		var levelName string
		switch *lv.filterLevel {
		case LogLevelInfo:
			levelName = "INFO"
		case LogLevelWarning:
			levelName = "WARN"
		case LogLevelError:
			levelName = "ERROR"
		case LogLevelSuccess:
			levelName = "OK"
		}
		filters = append(filters, "Level: "+levelName)
	}

	if lv.filterText != "" {
		filters = append(filters, "Text: "+lv.filterText)
	}

	filterStr := "Filters: " + strings.Join(filters, " | ")
	return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(filterStr)
}
