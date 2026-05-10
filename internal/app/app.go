package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/kairos-development/kairos-agent/internal/audit"
	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/journal"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	"github.com/kairos-development/kairos-agent/internal/storage"
	"github.com/kairos-development/kairos-agent/internal/vault"
	"github.com/sirupsen/logrus"
)

const (
	Version = "0.1.0"
)

// Application owns the bootstrapped agent subsystems.
type Application struct {
	ConfigManager *config.Manager
	Store         *storage.Store
	Vault         *vault.Store
	Audit         *audit.Logger
	Journal       *journal.Log
	Runtime       *runtime.Manager
	Logger        *slog.Logger
	StateDir      string
}

// Bootstrap creates the state directory, config, audit log, vault, journal, and storage.
func Bootstrap(ctx context.Context, stateDir string, vaultPassword string) (app *Application, err error) {
	var closers []func() error
	defer func() {
		if err == nil {
			return
		}
		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i]()
		}
	}()

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return nil, err
	}
	cfg := configManager.Current()
	if cfg.MemoryLimitBytes > 0 {
		runtime.ApplyMemoryLimit(cfg.MemoryLimitBytes)
	}
	auditLogger, err := audit.Open(cfg.Paths.AuditPath)
	if err != nil {
		return nil, err
	}
	closers = append(closers, auditLogger.Close)

	journalLog, err := journal.Open(cfg.Paths.JournalPath, cfg.JournalMaxSizeBytes)
	if err != nil {
		return nil, err
	}
	closers = append(closers, journalLog.Close)

	store, err := storage.Open(stateDir, Version)
	if err != nil {
		return nil, err
	}
	closers = append(closers, store.Close)

	vaultStore, err := vault.OpenOrCreate(cfg.Paths.VaultDB, vaultPassword, Version)
	if err != nil {
		return nil, err
	}
	closers = append(closers, vaultStore.Close)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	app = &Application{
		ConfigManager: configManager,
		Store:         store,
		Vault:         vaultStore,
		Audit:         auditLogger,
		Journal:       journalLog,
		Runtime:       runtime.NewManager(Version, logrus.New()),
		Logger:        logger,
		StateDir:      stateDir,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("NTP worker panic recovered", "panic", r, "stack", string(debug.Stack()))
			}
		}()
		runtime.RunNTPWorker(ctx, 15*time.Minute, func(context.Context) (time.Duration, error) {
			return 0, nil
		}, app.Runtime.UpdateNTPDrift)
	}()

	return app, nil
}

// Close closes application resources.
func (a *Application) Close() error {
	var errs []string
	if a.Journal != nil {
		if err := a.Journal.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if a.Audit != nil {
		if err := a.Audit.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if a.Store != nil {
		if err := a.Store.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if a.Vault != nil {
		if err := a.Vault.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
