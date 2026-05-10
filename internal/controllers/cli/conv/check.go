package conv

import (
	"fmt"
	"strings"

	"github.com/kairos-development/kairos-agent/internal/controllers/cli/dto"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
)

// CheckOutputToDTO converts a service check report into the CLI model.
func CheckOutputToDTO(output serviceagent.CheckOutput) dto.CheckOutput {
	return dto.CheckOutput{
		SchemaVersion:   output.SchemaVersion,
		Telemetry:       string(output.Telemetry.Profile),
		AllowedOutbound: append([]string(nil), output.AllowedOutbound...),
		Mode:            string(output.Status.Mode),
		Connectivity:    string(output.Status.Connectivity),
		License:         string(output.Status.License),
		Integrity:       string(output.Status.Integrity),
		HaltReason:      output.Status.HaltReason,
	}
}

// FormatCheckOutput renders the CLI check model.
func FormatCheckOutput(output dto.CheckOutput) string {
	return fmt.Sprintf(
		"schema_version=%d\ntelemetry=%s\noutbound=%s\nmode=%s\nconnectivity=%s\nlicense=%s\nintegrity=%s\nhalt_reason=%s\n",
		output.SchemaVersion,
		output.Telemetry,
		strings.Join(output.AllowedOutbound, ","),
		output.Mode,
		output.Connectivity,
		output.License,
		output.Integrity,
		output.HaltReason,
	)
}
