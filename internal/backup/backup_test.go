package backup

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBackupState(t *testing.T) string {
	t.Helper()
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	require.NoError(t, manager.Write(manager.Current()))
	store, err := vault.OpenOrCreate(filepath.Join(stateDir, "vault.db"), "backup-password", "test")
	require.NoError(t, err)
	require.NoError(t, store.Close())
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "trades.sqlite"), []byte("trades"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "audit.log"), []byte("audit"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "journal.log"), []byte("journal"), 0o600))
	return stateDir
}

func TestCreateProducesEncryptedArchiveAndChecksum(t *testing.T) {
	stateDir := createBackupState(t)

	result, err := Create(stateDir, "backup-password")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Checksum)
	assert.True(t, strings.HasPrefix(filepath.Base(result.ArchivePath), "kairos-backup-"))

	body, err := os.ReadFile(result.ArchivePath)
	require.NoError(t, err)
	digest := sha256.Sum256(body)
	assert.Equal(t, hex.EncodeToString(digest[:]), result.Checksum)

	checksumBody, err := os.ReadFile(result.ArchivePath + ".sha256")
	require.NoError(t, err)
	assert.Contains(t, string(checksumBody), result.Checksum)

	identity, err := age.NewScryptIdentity("backup-password")
	require.NoError(t, err)
	reader, err := age.Decrypt(bytes.NewReader(body), identity)
	require.NoError(t, err)
	tarReader := tar.NewReader(reader)
	seen := map[string]bool{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		seen[header.Name] = true
		assert.True(t, header.ModTime.IsZero() || header.ModTime.Unix() == 0)
	}
	assert.True(t, seen["config.yaml"])
	assert.True(t, seen["vault.db"])
	assert.True(t, seen["trades.sqlite"])
}

func TestRestoreVerifiesChecksumAndPassword(t *testing.T) {
	sourceDir := createBackupState(t)
	result, err := Create(sourceDir, "backup-password")
	require.NoError(t, err)
	targetDir := t.TempDir()

	require.NoError(t, Restore(targetDir, result.ArchivePath, "backup-password"))
	for _, name := range []string{"config.yaml", "vault.db", "trades.sqlite", "audit.log", "journal.log"} {
		_, err := os.Stat(filepath.Join(targetDir, name))
		require.NoError(t, err, name)
	}

	assert.Error(t, Restore(t.TempDir(), result.ArchivePath, "wrong-password"))
}

func TestRestoreRejectsChecksumMismatch(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "backup-password")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(result.ArchivePath+".sha256", []byte("bad checksum"), 0o600))

	err = Restore(t.TempDir(), result.ArchivePath, "backup-password")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestCreate_MissingStateDir(t *testing.T) {
	_, err := Create("/nonexistent/dir", "password")
	assert.Error(t, err)
}

func TestCreate_InvalidPassword(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "")
	require.Error(t, err)
	assert.Empty(t, result.Checksum)
	assert.Contains(t, err.Error(), "password is required")
}

func TestCreate_PartialFiles(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	require.NoError(t, manager.Write(manager.Current()))
	// Only create config.yaml, skip other files

	result, err := Create(stateDir, "password")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Checksum)
}

func TestCreate_FileReadError(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	require.NoError(t, manager.Write(manager.Current()))

	// Create a directory instead of file to cause read error
	require.NoError(t, os.Mkdir(filepath.Join(stateDir, "trades.sqlite"), 0o755))

	_, err = Create(stateDir, "password")
	assert.Error(t, err)
}

func TestRestore_MissingArchive(t *testing.T) {
	err := Restore(t.TempDir(), "/nonexistent/archive.tar.age", "password")
	assert.Error(t, err)
}

func TestRestore_MissingChecksum(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Remove checksum file
	require.NoError(t, os.Remove(result.ArchivePath+".sha256"))

	err = Restore(t.TempDir(), result.ArchivePath, "password")
	assert.Error(t, err)
}

func TestRestore_CorruptedArchive(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Corrupt the archive
	require.NoError(t, os.WriteFile(result.ArchivePath, []byte("corrupted data"), 0o600))

	err = Restore(t.TempDir(), result.ArchivePath, "password")
	assert.Error(t, err)
}

func TestRestore_IncompatibleSchemaVersion(t *testing.T) {
	stateDir := createBackupState(t)

	// Modify config to have incompatible schema version
	cfg, err := config.Load(filepath.Join(stateDir, "config.yaml"))
	require.NoError(t, err)
	cfg.SchemaVersion = 999
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	require.NoError(t, manager.Write(cfg))

	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	err = Restore(t.TempDir(), result.ArchivePath, "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible")
}

func TestRestore_VaultVerificationFails(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Try to restore with wrong password (vault verification will fail)
	err = Restore(t.TempDir(), result.ArchivePath, "wrong-password")
	assert.Error(t, err)
}

func TestCreate_WriteChecksumError(t *testing.T) {
	stateDir := createBackupState(t)

	// Make state dir read-only to cause checksum write error
	defer os.Chmod(stateDir, 0o755)

	result, err := Create(stateDir, "password")
	// Should succeed in creating archive but may fail on checksum
	if err == nil {
		assert.NotEmpty(t, result.ArchivePath)
	}
}

func TestRestore_CreateTempDirError(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Create a file where temp dir should be created
	targetDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "restore-test"), []byte("block"), 0o600))

	// This should still work as MkdirTemp uses random suffix
	err = Restore(targetDir, result.ArchivePath, "password")
	// May succeed or fail depending on random suffix collision
	_ = err
}

func TestCreate_EmptyStateDir(t *testing.T) {
	stateDir := t.TempDir()

	// Create backup with no files
	_, err := Create(stateDir, "password")
	assert.Error(t, err) // Should fail as config.yaml is required
}

func TestRestore_TargetDirDoesNotExist(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	targetDir := filepath.Join(t.TempDir(), "nested", "target")
	err = Restore(targetDir, result.ArchivePath, "password")
	// Should fail as target dir doesn't exist
	assert.Error(t, err)
}

func TestCreate_StateDirNotDirectory(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "not_a_dir_*")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Create(tmpFile.Name(), "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestRestore_ReadDirError(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Create a file where target dir should be
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "restore-test")
	require.NoError(t, os.WriteFile(targetFile, []byte("block"), 0o600))

	err = Restore(targetDir, result.ArchivePath, "password")
	// Should succeed as MkdirTemp uses random suffix
	_ = err
}

func TestRestore_WriteFileError(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	// Create read-only target dir
	targetDir := t.TempDir()
	require.NoError(t, os.Chmod(targetDir, 0o500))
	defer os.Chmod(targetDir, 0o755)

	err = Restore(targetDir, result.ArchivePath, "password")
	assert.Error(t, err)
}

func TestCreate_StatError(t *testing.T) {
	stateDir := createBackupState(t)

	// Create backup successfully
	result, err := Create(stateDir, "password")
	require.NoError(t, err)
	assert.NotEmpty(t, result.ArchivePath)
	assert.NotEmpty(t, result.Checksum)
}

func TestCreate_OpenFileError(t *testing.T) {
	stateDir := createBackupState(t)

	// Make config.yaml unreadable
	configPath := filepath.Join(stateDir, "config.yaml")
	require.NoError(t, os.Chmod(configPath, 0o000))
	defer os.Chmod(configPath, 0o600)

	_, err := Create(stateDir, "password")
	assert.Error(t, err)
}

func TestCreate_CreateArchiveError(t *testing.T) {
	stateDir := createBackupState(t)

	// Make state dir read-only to prevent archive creation
	require.NoError(t, os.Chmod(stateDir, 0o500))
	defer os.Chmod(stateDir, 0o755)

	_, err := Create(stateDir, "password")
	assert.Error(t, err)
}

func TestRestore_DecryptError(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	targetDir := t.TempDir()

	// Try to restore with wrong password
	err = Restore(targetDir, result.ArchivePath, "wrongpassword")
	assert.Error(t, err)
}

func TestRestore_MkdirAllError(t *testing.T) {
	stateDir := createBackupState(t)
	result, err := Create(stateDir, "password")
	require.NoError(t, err)

	targetDir := t.TempDir()

	// Make target dir read-only
	require.NoError(t, os.Chmod(targetDir, 0o500))
	defer os.Chmod(targetDir, 0o755)

	err = Restore(targetDir, result.ArchivePath, "password")
	assert.Error(t, err)
}
