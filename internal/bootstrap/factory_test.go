package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

func TestFactoryOpenAndContainerClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	container, err := Factory{StateDir: t.TempDir(), VaultPassword: "factory-password"}.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, container.Application)
	require.NotNil(t, container.AgentService)
	require.NotNil(t, container.TradingGate)
	_, err = os.Stat(filepath.Join(container.Application.StateDir, "runtime-status.json"))
	require.NoError(t, err)
	require.NoError(t, container.Close())
}

func TestContainerCloseNil(t *testing.T) {
	var container *Container
	require.NoError(t, container.Close())
	require.NoError(t, (&Container{}).Close())
}

func TestFactoryOpenRestoresUncleanActiveSnapshotAsHalted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateDir := t.TempDir()
	container, err := Factory{StateDir: stateDir, VaultPassword: "factory-password"}.Open(ctx)
	require.NoError(t, err)
	require.NoError(t, container.AgentService.StartPaper(ctx))
	require.NoError(t, container.Close())

	restored, err := Factory{StateDir: stateDir, VaultPassword: "factory-password"}.Open(ctx)
	require.NoError(t, err)
	defer restored.Close()

	status, err := restored.AgentService.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, entity.RunModeHalted, status.Mode)
	require.Equal(t, "unclean restart from paper_trading", status.HaltReason)
}
