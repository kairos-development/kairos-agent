package conv

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultStateDir returns the default Kairos state directory.
func DefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kairos"
	}
	return filepath.Join(home, ".kairos")
}

// DefaultExportFilename returns the default export filename for a UTC timestamp.
func DefaultExportFilename(now time.Time) string {
	return fmt.Sprintf("export-%s.csv", now.UTC().Format("20060102T150405Z"))
}
