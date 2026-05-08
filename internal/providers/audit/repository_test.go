package auditprovider

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryAcceptDisclaimer(t *testing.T) {
	logger, err := audit.Open(filepath.Join(t.TempDir(), "audit.log"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, logger.Close()) })
	repo := New(logger, "test-version")
	ctx := context.Background()

	accepted, err := repo.DisclaimerAccepted(ctx)
	require.NoError(t, err)
	assert.False(t, accepted)

	require.NoError(t, repo.AcceptDisclaimer(ctx, "tui"))
	accepted, err = repo.DisclaimerAccepted(ctx)
	require.NoError(t, err)
	assert.True(t, accepted)
}
