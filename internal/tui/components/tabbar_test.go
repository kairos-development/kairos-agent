package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTabBar(t *testing.T) {
	tb := NewTabBar()

	require.NotNil(t, tb)
	assert.Equal(t, 0, tb.activeTab)
	assert.Len(t, tb.tabs, 6)
	assert.Equal(t, 80, tb.width)
	assert.Equal(t, "F1", tb.tabs[0].Key)
	assert.Equal(t, "Dashboard", tb.tabs[0].Label)
}

func TestTabBar_SetActiveTab(t *testing.T) {
	tests := []struct {
		name      string
		index     int
		wantIndex int
	}{
		{
			name:      "valid index 0",
			index:     0,
			wantIndex: 0,
		},
		{
			name:      "valid index 3",
			index:     3,
			wantIndex: 3,
		},
		{
			name:      "valid index 5",
			index:     5,
			wantIndex: 5,
		},
		{
			name:      "negative index",
			index:     -1,
			wantIndex: 0, // Should not change
		},
		{
			name:      "index too large",
			index:     10,
			wantIndex: 0, // Should not change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTabBar()
			tb.SetActiveTab(tt.index)
			assert.Equal(t, tt.wantIndex, tb.activeTab)
		})
	}
}

func TestTabBar_SetWidth(t *testing.T) {
	tb := NewTabBar()
	tb.SetWidth(120)
	assert.Equal(t, 120, tb.width)
}

func TestTabBar_Render(t *testing.T) {
	tests := []struct {
		name      string
		activeTab int
		width     int
	}{
		{
			name:      "dashboard active",
			activeTab: 0,
			width:     80,
		},
		{
			name:      "market active",
			activeTab: 1,
			width:     80,
		},
		{
			name:      "logs active",
			activeTab: 4,
			width:     80,
		},
		{
			name:      "settings active",
			activeTab: 5,
			width:     80,
		},
		{
			name:      "wide terminal",
			activeTab: 0,
			width:     200,
		},
		{
			name:      "narrow terminal",
			activeTab: 0,
			width:     60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTabBar()
			tb.SetActiveTab(tt.activeTab)
			tb.SetWidth(tt.width)

			rendered := tb.Render()

			require.NotEmpty(t, rendered)

			// Verify all tab labels are present
			assert.Contains(t, rendered, "Dashboard")
			assert.Contains(t, rendered, "Market")
			assert.Contains(t, rendered, "Positions")
			assert.Contains(t, rendered, "Balance")
			assert.Contains(t, rendered, "Logs")
			assert.Contains(t, rendered, "Settings")

			// Verify help text is present
			if tt.activeTab == 4 {
				// Logs view has different help text
				assert.Contains(t, rendered, "Filter")
			} else {
				assert.Contains(t, rendered, "Help")
			}
		})
	}
}

func TestTabBar_Render_AllTabs(t *testing.T) {
	tb := NewTabBar()

	// Test rendering each tab as active
	for i := 0; i < 6; i++ {
		tb.SetActiveTab(i)
		rendered := tb.Render()
		assert.NotEmpty(t, rendered)
		// Should contain all tab labels regardless of which is active
		assert.Contains(t, rendered, "Dashboard")
		assert.Contains(t, rendered, "Market")
		assert.Contains(t, rendered, "Positions")
		assert.Contains(t, rendered, "Balance")
		assert.Contains(t, rendered, "Logs")
		assert.Contains(t, rendered, "Settings")
	}
}

func TestTabBar_Render_ContextSensitiveHelp(t *testing.T) {
	tb := NewTabBar()

	// Logs view (index 4) should show filter help
	tb.SetActiveTab(4)
	rendered := tb.Render()
	assert.Contains(t, rendered, "Filter")
	assert.Contains(t, rendered, "PgUp/Dn")

	// Other views should show general help
	tb.SetActiveTab(0)
	rendered = tb.Render()
	assert.Contains(t, rendered, "Help")
	assert.Contains(t, rendered, "Commands")
}

func TestTabBar_Render_NoEmptyOutput(t *testing.T) {
	tb := NewTabBar()

	for i := 0; i < 6; i++ {
		tb.SetActiveTab(i)
		rendered := tb.Render()
		assert.NotEmpty(t, rendered)
		assert.NotEqual(t, "", strings.TrimSpace(rendered))
	}
}
