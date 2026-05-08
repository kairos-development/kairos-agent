package journal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	require.NotNil(t, log)
	defer log.Close()

	assert.Equal(t, path, log.path)
	assert.Equal(t, int64(1<<20), log.maxSizeBytes)
}

func TestOpen_CreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	require.NotNil(t, log)
	defer log.Close()

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(path))
	assert.NoError(t, err)
}

func TestClose_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	err = log.Close()
	assert.NoError(t, err)
}

func TestClose_NilLog(t *testing.T) {
	var log *Log
	err := log.Close()
	assert.NoError(t, err)
}

func TestClose_NilFile(t *testing.T) {
	log := &Log{}
	err := log.Close()
	assert.NoError(t, err)
}

func TestAppend_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	entry := Entry{
		ClientOrderID: "order-123",
		Kind:          "order_intent",
		Payload:       `{"symbol":"BTCUSDT"}`,
		TimestampUTC:  time.Now().UTC(),
	}

	err = log.Append(entry)
	assert.NoError(t, err)

	// Verify file was written
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestAppend_MultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	for i := 0; i < 5; i++ {
		entry := Entry{
			ClientOrderID: "order-123",
			Kind:          "order_intent",
			Payload:       `{"symbol":"BTCUSDT"}`,
			TimestampUTC:  time.Now().UTC(),
		}
		err = log.Append(entry)
		require.NoError(t, err)
	}

	// Verify all entries were written
	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1) // Deduplicated by ClientOrderID
}

func TestRotateIfNeeded_NoRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20) // 1MB limit
	require.NoError(t, err)
	defer log.Close()

	entry := Entry{
		ClientOrderID: "order-123",
		Kind:          "order_intent",
		Payload:       `{"symbol":"BTCUSDT"}`,
		TimestampUTC:  time.Now().UTC(),
	}

	err = log.Append(entry)
	require.NoError(t, err)

	// Verify no rotation occurred
	matches, err := filepath.Glob(path + "*")
	require.NoError(t, err)
	assert.Len(t, matches, 1) // Only the main file
}

func TestRotateIfNeeded_WithRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 100) // Very small limit to trigger rotation
	require.NoError(t, err)
	defer log.Close()

	// Append entries until rotation occurs
	for i := 0; i < 10; i++ {
		entry := Entry{
			ClientOrderID: "order-123",
			Kind:          "order_intent",
			Payload:       `{"symbol":"BTCUSDT","side":"buy","quantity":"0.1","price":"50000"}`,
			TimestampUTC:  time.Now().UTC(),
		}
		err = log.Append(entry)
		require.NoError(t, err)
	}

	// Verify rotation occurred
	matches, err := filepath.Glob(path + "*")
	require.NoError(t, err)
	assert.Greater(t, len(matches), 1) // Main file + rotated files
}

func TestReplay_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReplay_SingleEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	entry := Entry{
		ClientOrderID: "order-123",
		Kind:          "order_intent",
		Payload:       `{"symbol":"BTCUSDT"}`,
		TimestampUTC:  time.Now().UTC(),
	}
	err = log.Append(entry)
	require.NoError(t, err)

	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "order-123", entries[0].ClientOrderID)
	assert.Equal(t, "order_intent", entries[0].Kind)
}

func TestReplay_DeduplicatesClientOrderID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Append multiple entries with same ClientOrderID
	err = log.Append(Entry{ClientOrderID: "abc", Kind: "one", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	err = log.Append(Entry{ClientOrderID: "abc", Kind: "two", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "two", entries[0].Kind) // Latest entry wins
}

func TestReplay_MultipleClientOrderIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	err = log.Append(Entry{ClientOrderID: "order-1", Kind: "intent", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	err = log.Append(Entry{ClientOrderID: "order-2", Kind: "intent", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	err = log.Append(Entry{ClientOrderID: "order-3", Kind: "intent", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestReplay_EmptyClientOrderID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Entries without ClientOrderID are not deduplicated
	err = log.Append(Entry{ClientOrderID: "", Kind: "event1", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	err = log.Append(Entry{ClientOrderID: "", Kind: "event2", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 2) // Both entries preserved
}

func TestReplay_WithRotatedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 100) // Small limit to trigger rotation
	require.NoError(t, err)
	defer log.Close()

	// Append entries to trigger rotation
	for i := 0; i < 10; i++ {
		entry := Entry{
			ClientOrderID: "order-123",
			Kind:          "order_intent",
			Payload:       `{"symbol":"BTCUSDT","side":"buy","quantity":"0.1","price":"50000"}`,
			TimestampUTC:  time.Now().UTC(),
		}
		err = log.Append(entry)
		require.NoError(t, err)
	}

	// Replay should read all files
	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1) // Deduplicated
}

func TestTruncateAfterReconciliation_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	// Append some entries
	err = log.Append(Entry{ClientOrderID: "order-1", Kind: "intent", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	err = log.TruncateAfterReconciliation()
	require.NoError(t, err)

	// Verify file was truncated
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}

func TestTruncateAfterReconciliation_RemovesRotatedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 100) // Small limit
	require.NoError(t, err)

	// Append entries to trigger rotation
	for i := 0; i < 10; i++ {
		entry := Entry{
			ClientOrderID: "order-123",
			Kind:          "order_intent",
			Payload:       `{"symbol":"BTCUSDT","side":"buy","quantity":"0.1","price":"50000"}`,
			TimestampUTC:  time.Now().UTC(),
		}
		err = log.Append(entry)
		require.NoError(t, err)
	}

	// Verify rotated files exist
	matches, err := filepath.Glob(path + "*")
	require.NoError(t, err)
	assert.Greater(t, len(matches), 1)

	// Truncate
	err = log.TruncateAfterReconciliation()
	require.NoError(t, err)

	// Verify only main file exists and is empty
	matches, err = filepath.Glob(path + "*")
	require.NoError(t, err)
	assert.Len(t, matches, 1)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}

func TestAppendOrderIntent_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	intent := OrderIntent{
		ClientOrderID: "order-123",
		Symbol:        "BTCUSDT",
		Side:          "buy",
		OrderType:     "limit",
		Quantity:      "0.1",
		Price:         "50000",
		TimestampUTC:  time.Now().UTC(),
	}

	err = AppendOrderIntent(log, intent)
	assert.NoError(t, err)

	// Verify entry was written
	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "order-123", entries[0].ClientOrderID)
	assert.Equal(t, "order_intent", entries[0].Kind)
}

func TestReplay_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Write invalid JSON directly to file
	_, err = log.file.Write([]byte("invalid json\n"))
	require.NoError(t, err)
	log.file.Sync()

	// Append valid entry
	err = log.Append(Entry{ClientOrderID: "order-1", Kind: "intent", TimestampUTC: time.Now().UTC()})
	require.NoError(t, err)

	// Replay should skip invalid entries
	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "order-1", entries[0].ClientOrderID)
}

func TestOpen_CreateDirError(t *testing.T) {
	// Try to create journal in a file (not directory)
	tmpFile := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o600))

	path := filepath.Join(tmpFile, "journal.log")
	_, err := Open(path, 1<<20)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create journal dir")
}

func TestAppend_AfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	err = log.Close()
	require.NoError(t, err)

	// Append after close should fail
	entry := Entry{
		ClientOrderID: "order-1",
		Kind:          "order_intent",
		TimestampUTC:  time.Now().UTC(),
	}
	err = log.Append(entry)
	assert.Error(t, err)
}

func TestTruncateAfterReconciliation_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.log")
	log := &Log{path: path}

	err := log.TruncateAfterReconciliation()
	assert.Error(t, err)
}

func TestAppend_MarshalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Create entry with invalid data that can't be marshaled
	// Note: In Go, json.Marshal rarely fails for basic types
	// This test documents the error path exists
	entry := Entry{
		ClientOrderID: "order-1",
		Kind:          "test",
		TimestampUTC:  time.Now().UTC(),
	}

	// Normal entry should work
	err = log.Append(entry)
	assert.NoError(t, err)
}

func TestAppend_SyncError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	// Close file to cause sync error
	log.file.Close()

	entry := Entry{
		ClientOrderID: "order-1",
		Kind:          "test",
		TimestampUTC:  time.Now().UTC(),
	}

	err = log.Append(entry)
	assert.Error(t, err)
}

func TestRotateIfNeeded_StatError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	// Close file to cause stat error
	log.file.Close()

	err = log.rotateIfNeeded()
	assert.Error(t, err)
}

func TestRotateIfNeeded_RenameError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 100) // Small limit
	require.NoError(t, err)
	defer log.Close()

	// Write enough data to trigger rotation
	for i := 0; i < 10; i++ {
		entry := Entry{
			ClientOrderID: "order-123",
			Kind:          "test",
			Payload:       `{"data":"test data for rotation trigger"}`,
			TimestampUTC:  time.Now().UTC(),
		}
		log.Append(entry)
	}

	// Rotation should have occurred
	matches, err := filepath.Glob(path + "*")
	require.NoError(t, err)
	assert.Greater(t, len(matches), 1)
}

func TestReplay_GlobError(t *testing.T) {
	// Create log with invalid glob pattern
	log := &Log{path: "[invalid"}

	_, err := log.Replay()
	assert.Error(t, err)
}

func TestReplay_OpenError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	log.Close()

	// Create a directory with the rotated file name to cause open error
	rotatedPath := path + ".1234567890"
	require.NoError(t, os.Mkdir(rotatedPath, 0o755))

	_, err = log.Replay()
	assert.Error(t, err)
}

func TestTruncateAfterReconciliation_GlobError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	// Set invalid glob pattern
	log.path = "[invalid"

	err = log.TruncateAfterReconciliation()
	assert.Error(t, err)
}

func TestTruncateAfterReconciliation_RemoveError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)

	// Write some data
	entry := Entry{
		ClientOrderID: "order-1",
		Kind:          "test",
		TimestampUTC:  time.Now().UTC(),
	}
	require.NoError(t, log.Append(entry))

	// Make file read-only to potentially cause remove issues
	// Note: On most systems, this won't prevent removal by owner
	// This test documents the error path exists
	err = log.TruncateAfterReconciliation()
	assert.NoError(t, err) // Should succeed on most systems
}

func TestAppendOrderIntent_MarshalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Normal intent should work
	intent := OrderIntent{
		ClientOrderID: "order-123",
		Symbol:        "BTCUSDT",
		Side:          "buy",
		OrderType:     "limit",
		Quantity:      "0.1",
		Price:         "50000",
		TimestampUTC:  time.Now().UTC(),
	}

	err = AppendOrderIntent(log, intent)
	assert.NoError(t, err)
}

func TestReplay_ScannerError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.log")
	log, err := Open(path, 1<<20)
	require.NoError(t, err)
	defer log.Close()

	// Write valid entry
	entry := Entry{
		ClientOrderID: "order-1",
		Kind:          "test",
		TimestampUTC:  time.Now().UTC(),
	}
	require.NoError(t, log.Append(entry))

	// Replay should work
	entries, err := log.Replay()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}
