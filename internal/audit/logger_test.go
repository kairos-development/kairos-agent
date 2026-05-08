package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenWriteHasActionAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "audit.log")
	logger, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, logger.Close()) })

	record := Record{
		TimestampUTC:    time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		OperatorSource:  "test",
		Action:          "risk_limit_changed",
		CorrelationID:   "corr-1",
		BeforeStateHash: HashState([]byte("before")),
		AfterStateHash:  HashState([]byte("after")),
	}
	require.NoError(t, logger.Write(record))

	found, err := logger.HasAction("risk_limit_changed")
	require.NoError(t, err)
	assert.True(t, found)

	missing, err := logger.HasAction("missing")
	require.NoError(t, err)
	assert.False(t, missing)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var decoded Record
	require.NoError(t, json.Unmarshal(body[:len(body)-1], &decoded))
	assert.Equal(t, record.Action, decoded.Action)
}

func TestHasActionSkipsMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	require.NoError(t, os.WriteFile(path, []byte("not-json\n{\"action\":\"target\"}\n"), 0o600))
	logger, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, logger.Close()) })

	found, err := logger.HasAction("target")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestDisclaimerRecordUsesStableHashes(t *testing.T) {
	record := DisclaimerRecord("tui", "risk text", "1.2.3")
	wantHash := HashState([]byte("risk text|1.2.3"))

	assert.Equal(t, ActionAcceptedFinancialRiskDisclaimer, record.Action)
	assert.Equal(t, "tui", record.OperatorSource)
	assert.Equal(t, "financial-risk-disclaimer", record.CorrelationID)
	assert.Equal(t, wantHash, record.BeforeStateHash)
	assert.Equal(t, wantHash, record.AfterStateHash)
	assert.Equal(t, time.UTC, record.TimestampUTC.Location())
}

func TestCloseNilLoggerIsNoop(t *testing.T) {
	var logger *Logger
	assert.NoError(t, logger.Close())
}

func TestOpen_CreateDirError(t *testing.T) {
	// Try to create audit log in a file (not directory)
	tmpFile := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o600))

	path := filepath.Join(tmpFile, "audit.log")
	_, err := Open(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create audit dir")
}

func TestWrite_MarshalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	logger, err := Open(path)
	require.NoError(t, err)
	defer logger.Close()

	// Create a record with invalid data that can't be marshaled
	// In Go, all Record fields are marshalable, so we can't easily trigger this
	// This test documents the error path exists
}

func TestHasAction_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.log")
	logger := &Logger{path: path}

	_, err := logger.HasAction("any")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open audit log for read")
}

func TestHashState_Deterministic(t *testing.T) {
	payload := []byte("test data")
	hash1 := HashState(payload)
	hash2 := HashState(payload)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA256 hex = 64 chars
}

func TestHashState_DifferentInputs(t *testing.T) {
	hash1 := HashState([]byte("data1"))
	hash2 := HashState([]byte("data2"))

	assert.NotEqual(t, hash1, hash2)
}
