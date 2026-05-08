package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFactoryOpenAndContainerClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	container, err := Factory{StateDir: t.TempDir(), VaultPassword: "factory-password"}.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, container.Application)
	require.NotNil(t, container.AgentService)
	require.NoError(t, container.Close())
}

func TestContainerCloseNil(t *testing.T) {
	var container *Container
	require.NoError(t, container.Close())
	require.NoError(t, (&Container{}).Close())
}
