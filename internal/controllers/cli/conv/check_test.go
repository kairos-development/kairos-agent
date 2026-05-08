package conv

import (
	"strings"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/stretchr/testify/assert"
)

func TestCheckOutputToDTOAndFormat(t *testing.T) {
	output := serviceagent.CheckOutput{
		SchemaVersion:   2,
		Telemetry:       entity.TelemetryConsent{Enabled: false, Profile: entity.TelemetryProfileOff},
		AllowedOutbound: []string{"license", "health"},
		Status: entity.RuntimeStatus{
			Mode:         entity.RunModePaperTrading,
			Connectivity: entity.ConnectivityStateConnected,
			License:      entity.LicenseStateDemo,
			Integrity:    entity.IntegrityStateTrusted,
		},
	}

	dto := CheckOutputToDTO(output)
	assert.Equal(t, 2, dto.SchemaVersion)
	assert.Equal(t, []string{"license", "health"}, dto.AllowedOutbound)

	rendered := FormatCheckOutput(dto)
	for _, part := range []string{"schema_version=2", "telemetry=off", "outbound=license,health", "mode=paper_trading", "connectivity=connected", "license=demo", "integrity=trusted"} {
		assert.True(t, strings.Contains(rendered, part), rendered)
	}
}
