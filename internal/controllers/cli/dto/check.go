package dto

// CheckOutput is the CLI presentation model for kairos check output.
type CheckOutput struct {
	SchemaVersion   int
	Telemetry       string
	AllowedOutbound []string
	Mode            string
	Connectivity    string
	License         string
	Integrity       string
	HaltReason      string
}
