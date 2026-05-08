package storageprovider

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepositorySetsDefaults(t *testing.T) {
	store, err := storage.Open(t.TempDir(), "test")
	require.NoError(t, err)
	defer store.Close()
	manager, err := config.NewManager(t.TempDir())
	require.NoError(t, err)

	repo := New(store, manager, "/tmp/export")

	assert.Same(t, store, repo.store)
	assert.Same(t, manager, repo.config)
	assert.Equal(t, "/tmp/export", repo.defaultDir)
	assert.NotNil(t, repo.clock)
	assert.NotNil(t, repo.createWriter)
}

func TestDefaultExportFilenameUsesUTC(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 34, 56, 0, time.FixedZone("UTC+3", 3*60*60))
	assert.Equal(t, "export-20260429T093456Z.csv", defaultExportFilename(now))
}

func TestExportRowsFromStorageMapsAllFields(t *testing.T) {
	now := time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC)
	rows := []storage.ExportTradeRecord{{
		Date:        now,
		Pair:        "BTCUSDT",
		Side:        "buy",
		Size:        decimal.RequireFromString("0.1"),
		EntryPrice:  decimal.RequireFromString("50000"),
		ExitPrice:   decimal.RequireFromString("51000"),
		Fee:         decimal.RequireFromString("5"),
		RealizedPNL: decimal.RequireFromString("95"),
	}}

	out := exportRowsFromStorage(rows)
	require.Len(t, out, 1)
	assert.Equal(t, now, out[0].Date)
	assert.Equal(t, "BTCUSDT", out[0].Pair)
	assert.Equal(t, "buy", out[0].Side)
	assert.True(t, out[0].Size.Equal(rows[0].Size))
	assert.True(t, out[0].EntryPrice.Equal(rows[0].EntryPrice))
	assert.True(t, out[0].ExitPrice.Equal(rows[0].ExitPrice))
	assert.True(t, out[0].Fee.Equal(rows[0].Fee))
	assert.True(t, out[0].RealizedPNL.Equal(rows[0].RealizedPNL))
}

func TestRepositoryExportCSVExplicitDestinationWritesFile(t *testing.T) {
	stateDir := t.TempDir()
	store, err := storage.Open(stateDir, "test")
	require.NoError(t, err)
	defer store.Close()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	insertClosedTrade(t, store, time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC))

	outputPath := filepath.Join(stateDir, "out.csv")
	repo := New(store, manager, stateDir)
	output, err := repo.ExportCSV(context.Background(), serviceagent.ExportCSVInput{DestinationPath: outputPath})
	require.NoError(t, err)
	assert.Equal(t, outputPath, output.Path)

	body, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "BTCUSDT")
	assert.Contains(t, string(body), "95")
}

func TestRepositoryExportCSVDefaultDestinationUsesClock(t *testing.T) {
	stateDir := t.TempDir()
	store, err := storage.Open(stateDir, "test")
	require.NoError(t, err)
	defer store.Close()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(store, manager, stateDir)
	repo.clock = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }

	output, err := repo.ExportCSV(context.Background(), serviceagent.ExportCSVInput{})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(stateDir, "export-20260429T090000Z.csv"), output.Path)
	assert.FileExists(t, output.Path)
}

func TestRepositoryExportCSVPropagatesStoreAndWriterErrors(t *testing.T) {
	stateDir := t.TempDir()
	store, err := storage.Open(stateDir, "test")
	require.NoError(t, err)
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	repo := New(store, manager, stateDir)

	require.NoError(t, store.Close())
	_, err = repo.ExportCSV(context.Background(), serviceagent.ExportCSVInput{DestinationPath: filepath.Join(stateDir, "closed.csv")})
	require.Error(t, err)

	store, err = storage.Open(stateDir, "test")
	require.NoError(t, err)
	defer store.Close()
	repo = New(store, manager, stateDir)
	repo.createWriter = func(string) (*os.File, error) { return nil, errors.New("writer failed") }

	_, err = repo.ExportCSV(context.Background(), serviceagent.ExportCSVInput{DestinationPath: filepath.Join(stateDir, "out.csv")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writer failed")
}

func insertClosedTrade(t *testing.T, store *storage.Store, date time.Time) {
	t.Helper()
	err := store.WithCriticalTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO closed_trades (date_utc, pair, side, size, entry_price, exit_price, fee, realized_pnl)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			date.Format(time.RFC3339Nano),
			"BTCUSDT",
			"buy",
			"0.1",
			"50000",
			"51000",
			"5",
			"95",
		)
		return err
	})
	require.NoError(t, err)
}
