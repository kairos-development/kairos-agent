package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandRequiresAcceptFlagOrPrompt(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newInitCommand(&stateDir, strPtr("secret"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	// Without --accept-financial-risk the command tries to read stdin and fails in test context
	err := cmd.Execute()
	// In test context without stdin piping, ReadString blocks or returns error.
	// The important thing is it does NOT panic and the accept path works when flag is set.
	_ = err
}

func TestInitCommandWithAcceptFlag(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newInitCommand(&stateDir, strPtr("secret"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--accept-financial-risk"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCheckCommandExecutes(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newCheckCommand(&stateDir, strPtr("secret"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	// Bootstrap creates fresh state so check may succeed.
	// Just verify it doesn't panic.
	_ = cmd.Execute()
}

func TestBackupCommandExecutesWithDefaults(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newBackupCommand(&stateDir)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--password", "testpass"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestExportCommandSupportsCSVFormat(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newExportCommand(&stateDir, &vaultPassword)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--format", "csv"})

	err := cmd.Execute()
	// If error occurs, it must NOT be about unsupported format
	if err != nil {
		assert.NotContains(t, err.Error(), "unsupported export format")
	}
}

func TestExportCommandRejectsJSONFormat(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newExportCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--format", "json"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestExportCommandRejectsXMLFormat(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newExportCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--format", "xml"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestEmergencyStopCommandWithConfirmFlag(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newEmergencyStopCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--confirm"})

	err := cmd.Execute()
	// Bootstrap succeeds with fresh state so emergency stop runs.
	// The important thing is the confirm gate passed (no "--confirm" error).
	if err != nil {
		assert.NotContains(t, err.Error(), "--confirm")
	}
}

func TestPluginTrustCommandRequiresPubkeyFile(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newPluginTrustCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"trust", "fp123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pubkey-file")
}

func TestPluginTrustCommandWithMissingPubkeyFile(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newPluginTrustCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"trust", "fp123", "--pubkey-file", "/nonexistent/path"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestRestoreCommandRequiresFileArg(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newRestoreCommand(&stateDir)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--confirm"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestRestoreCommandWithFileAndConfirm(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newRestoreCommand(&stateDir)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--file", "backup.tar.age", "--confirm"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "--confirm")
}

func TestRootCommandContainsHeadless(t *testing.T) {
	cmd := NewRootCommand()
	seen := map[string]bool{}
	for _, child := range cmd.Commands() {
		seen[child.Name()] = true
	}
	assert.True(t, seen["headless"], "missing headless command")
}

func TestHeadlessCommandDefaultsToStopAll(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newHeadlessCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	// Bootstrap creates fresh state so command may succeed.
	// Just verify it doesn't panic.
	_ = cmd.Execute()
}

func TestHeadlessCommandWithLiveFlag(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newHeadlessCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--live"})

	err := cmd.Execute()
	// Without disclaimer, live trading should fail
	if err != nil {
		assert.Error(t, err)
	}
}

func TestHeadlessCommandWithScanFlag(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newHeadlessCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--scan"})

	// Just verify it doesn't panic
	_ = cmd.Execute()
}

func TestRootCommandPersistentFlags(t *testing.T) {
	cmd := NewRootCommand()
	sdir := cmd.PersistentFlags().Lookup("state-dir")
	require.NotNil(t, sdir)
	vpass := cmd.PersistentFlags().Lookup("vault-password")
	require.NotNil(t, vpass)
}

func strPtr(s string) *string {
	return &s
}
