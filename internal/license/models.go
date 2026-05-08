package license

// State identifies the locally evaluated license state.
type State string

const (
	StateDemo       State = "demo"
	StateLicensed   State = "licensed"
	StateGrace      State = "grace"
	StateRiskOnly   State = "risk_only"
	StateUnlicensed State = "unlicensed"
)
