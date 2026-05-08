package storageprovider

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/export"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/storage"
)

// Repository adapts trade storage and exports to service-layer interfaces.
type Repository struct {
	store        *storage.Store
	config       *config.Manager
	defaultDir   string
	clock        func() time.Time
	createWriter func(string) (*os.File, error)
}

// New creates a storage provider repository.
func New(store *storage.Store, config *config.Manager, defaultDir string) *Repository {
	return &Repository{
		store:        store,
		config:       config,
		defaultDir:   defaultDir,
		clock:        func() time.Time { return time.Now().UTC() },
		createWriter: os.Create,
	}
}

// ExportCSV writes a CSV export to the requested path.
func (r *Repository) ExportCSV(ctx context.Context, input serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error) {
	path := input.DestinationPath
	if path == "" {
		path = filepath.Join(r.defaultDir, defaultExportFilename(r.clock()))
	}
	rows, err := r.store.ListExportTrades(ctx)
	if err != nil {
		return serviceagent.ExportCSVOutput{}, err
	}
	file, err := r.createWriter(path)
	if err != nil {
		return serviceagent.ExportCSVOutput{}, err
	}
	defer file.Close()
	if err := export.WriteCSV(file, r.config.Current().ExportTimezone, exportRowsFromStorage(rows)); err != nil {
		return serviceagent.ExportCSVOutput{}, err
	}
	return serviceagent.ExportCSVOutput{Path: path}, nil
}

func defaultExportFilename(now time.Time) string {
	return "export-" + now.UTC().Format("20060102T150405Z") + ".csv"
}

func exportRowsFromStorage(rows []storage.ExportTradeRecord) []export.TradeRecord {
	out := make([]export.TradeRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, export.TradeRecord{
			Date:        row.Date,
			Pair:        row.Pair,
			Side:        row.Side,
			Size:        row.Size,
			EntryPrice:  row.EntryPrice,
			ExitPrice:   row.ExitPrice,
			Fee:         row.Fee,
			RealizedPNL: row.RealizedPNL,
		})
	}
	return out
}
