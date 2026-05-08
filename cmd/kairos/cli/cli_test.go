package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeCommand(t *testing.T, cmdArgs ...string) error {
	t.Helper()
	cmd := NewRootCommand()
	cmd.SetArgs(cmdArgs)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd.Execute()
}

func TestRootCommandHasExpectedSubcommands(t *testing.T) {
	cmd := NewRootCommand()
	assert.Equal(t, "kairos", cmd.Use)
	for _, name := range []string{"init", "check", "headless", "backup", "restore", "export", "emergency-stop", "plugin", "update", "version"} {
		_, _, err := cmd.Find([]string{name})
		assert.NoError(t, err, name)
	}
}

func TestVersionAndUpdateCommands(t *testing.T) {
	require.NoError(t, executeCommand(t, "version", "--short"))
	require.NoError(t, executeCommand(t, "version"))
	require.NoError(t, executeCommand(t, "update", "check", "--channel", "dev"))
	require.NoError(t, executeCommand(t, "update", "download", "--version", "1.0.1"))
	assert.Error(t, executeCommand(t, "update", "apply"))
}

func TestInitAndCheckCommands(t *testing.T) {
	stateDir := t.TempDir()
	require.NoError(t, executeCommand(t, "init", "--state-dir", stateDir, "--accept-disclaimer"))
	_, err := os.Stat(filepath.Join(stateDir, "config.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(stateDir, "disclaimer_accepted"))
	require.NoError(t, err)
	assert.Error(t, executeCommand(t, "init", "--state-dir", stateDir, "--accept-disclaimer"))
	require.NoError(t, executeCommand(t, "check", "--state-dir", stateDir, "--verbose"))
}

func TestHeadlessValidation(t *testing.T) {
	assert.Error(t, executeCommand(t, "headless"))
	assert.Error(t, executeCommand(t, "headless", "--paper", "--scan"))
	assert.Error(t, executeCommand(t, "headless", "--live"))
}

func TestEmergencyStopValidation(t *testing.T) {
	assert.Error(t, executeCommand(t, "emergency-stop"))
}

func TestPluginCommandsValidationAndList(t *testing.T) {
	stateDir := t.TempDir()
	_, err := config.NewManager(stateDir)
	require.NoError(t, err)
	assert.Error(t, executeCommand(t, "plugin", "trust", "abc", "--state-dir", stateDir))
	require.NoError(t, executeCommand(t, "plugin", "list", "--state-dir", stateDir))
}

func TestBackupRestoreExportValidation(t *testing.T) {
	stateDir := t.TempDir()
	backupPath := filepath.Join(stateDir, "backup.tar.gz.enc")
	require.NoError(t, executeCommand(t, "backup", "--state-dir", stateDir, "--output", backupPath, "--full=false", "--compress=false"))
	assert.FileExists(t, backupPath)
	assert.Error(t, executeCommand(t, "restore", "--state-dir", stateDir))
	assert.Error(t, executeCommand(t, "restore", "--state-dir", stateDir, "--confirm", "--file", filepath.Join(stateDir, "missing.tar.gz.enc")))
	assert.Error(t, executeCommand(t, "export", "--state-dir", stateDir, "--format", "json"))
	require.NoError(t, executeCommand(t, "export", "--state-dir", stateDir, "--output", filepath.Join(stateDir, "trades.csv")))
}

func TestEmergencyStopConfirmed(t *testing.T) {
	stateDir := t.TempDir()
	_, err := config.NewManager(stateDir)
	require.NoError(t, err)
	require.NoError(t, executeCommand(t, "emergency-stop", "--state-dir", stateDir, "--confirm"))
}

func TestGetStateDirUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stateDir, err := getStateDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".kairos"), stateDir)
}

func TestConfirmPromptReadsPositiveAndNegativeAnswers(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString("yes\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	os.Stdin = reader
	confirmed, err := confirmPrompt("continue?")
	require.NoError(t, err)
	assert.True(t, confirmed)

	reader, writer, err = os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString("no\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	os.Stdin = reader
	confirmed, err = confirmPrompt("continue?")
	require.NoError(t, err)
	assert.False(t, confirmed)
}
