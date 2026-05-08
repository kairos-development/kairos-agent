package conv

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultStateDir(t *testing.T) {
	assert.True(t, strings.HasSuffix(DefaultStateDir(), ".kairos"))
}

func TestDefaultExportFilenameUsesUTC(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 30, 45, 0, time.FixedZone("MSK", 3*60*60))
	assert.Equal(t, "export-20260504T123045Z.csv", DefaultExportFilename(now))
}
