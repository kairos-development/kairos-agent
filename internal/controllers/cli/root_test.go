package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommandBuildsExpectedCommandTree(t *testing.T) {
	cmd := NewRootCommand()
	require.NotNil(t, cmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())

	seen := map[string]bool{}
	for _, child := range cmd.Commands() {
		seen[strings.Fields(child.Use)[0]] = true
	}
	for _, name := range []string{"init", "check", "backup", "restore", "export", "emergency-stop", "plugin", "headless"} {
		assert.True(t, seen[name], "missing command %s", name)
	}
}

func TestRestoreCommandRequiresConfirmBeforeRestore(t *testing.T) {
	stateDir := t.TempDir()
	cmd := newRestoreCommand(&stateDir)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--file", "backup.tar.age"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestExportCommandRejectsUnsupportedFormatBeforeBootstrap(t *testing.T) {
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

func TestEmergencyStopCommandRequiresConfirmBeforeBootstrap(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newEmergencyStopCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestPluginTrustCommandValidatesArgsAndRequiredPubkey(t *testing.T) {
	stateDir := t.TempDir()
	vaultPassword := "secret"
	cmd := newPluginTrustCommand(&stateDir, &vaultPassword)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"trust"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")
}
