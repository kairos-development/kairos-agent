package audit

import "time"

// Record is the audit-layer append-only event model.
type Record struct {
	TimestampUTC    time.Time `json:"timestamp_utc"`
	OperatorSource  string    `json:"operator_source"`
	Action          string    `json:"action"`
	CorrelationID   string    `json:"correlation_id"`
	BeforeStateHash string    `json:"before_state_hash"`
	AfterStateHash  string    `json:"after_state_hash"`
}
