package runtimeprovider

import (
	"context"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryMapsRuntimeState(t *testing.T) {
	manager := runtime.NewManager("test", logrus.New())
	repo := New(manager)
	ctx := context.Background()

	require.NoError(t, repo.Transition(ctx, entity.RunModeLiveTrading, "test"))
	require.NoError(t, repo.SetLicense(ctx, entity.LicenseStateLicensed))
	require.NoError(t, repo.SetConnectivity(ctx, entity.ConnectivityStateNetworkWait))

	status, err := repo.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeLiveTrading, status.Mode)
	assert.Equal(t, entity.LicenseStateLicensed, status.License)
	assert.Equal(t, entity.ConnectivityStateNetworkWait, status.Connectivity)
}
