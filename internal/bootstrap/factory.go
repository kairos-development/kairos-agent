package bootstrap

import (
	"context"
	"path/filepath"

	"github.com/kairos-development/kairos-agent/internal/app"
	auditprovider "github.com/kairos-development/kairos-agent/internal/providers/audit"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	storageprovider "github.com/kairos-development/kairos-agent/internal/providers/storage"
	vaultprovider "github.com/kairos-development/kairos-agent/internal/providers/vault"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/service/tradinggate"
)

// Container owns the bootstrapped application and layered services.
type Container struct {
	Application  *app.Application
	AgentService serviceagent.Service
	TradingGate  *tradinggate.Gate
}

// Close closes the bootstrapped resources.
func (c *Container) Close() error {
	if c == nil || c.Application == nil {
		return nil
	}
	return c.Application.Close()
}

// Factory creates layered application containers.
type Factory struct {
	StateDir      string
	VaultPassword string
}

// Open bootstraps the low-level application and layered service graph.
func (f Factory) Open(ctx context.Context) (*Container, error) {
	application, err := app.Bootstrap(ctx, f.StateDir, f.VaultPassword)
	if err != nil {
		return nil, err
	}
	runtimeRepository := runtimeprovider.New(
		application.Runtime,
		runtimeprovider.WithSnapshotStore(runtimeprovider.NewSnapshotStore(filepath.Join(application.StateDir, "runtime-status.json"))),
	)
	configRepository := configprovider.New(application.ConfigManager)
	auditRepository := auditprovider.New(application.Audit, app.Version)
	storageRepository := storageprovider.New(application.Store, application.ConfigManager, application.StateDir)
	vaultRepository := vaultprovider.New(application.Vault)
	service := serviceagent.New(runtimeRepository, configRepository, auditRepository, storageRepository, vaultRepository)
	gate := tradinggate.New(runtimeRepository, tradinggate.DefaultPolicy())
	return &Container{Application: application, AgentService: service, TradingGate: gate}, nil
}
