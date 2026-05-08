package integrity

// Status identifies runtime self-integrity state.
type Status string

const (
	StatusTrusted  Status = "trusted"
	StatusDegraded Status = "degraded"
)
