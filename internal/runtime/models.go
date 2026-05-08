package runtime

import (
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

// Status captures the mutable runtime state of the agent per instruction 8.3.
type Status struct {
	Mode              entity.RunMode
	Connectivity      entity.ConnectivityState
	License           entity.LicenseState
	Integrity         entity.IntegrityState
	NTPDrift          time.Duration
	NewEntriesBlocked bool
	Banner            string
	AnalyticsDegraded bool
	LastUpdatedAtUTC  time.Time
}
