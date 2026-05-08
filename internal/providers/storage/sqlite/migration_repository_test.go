package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationRepositoryRecordGetAndList(t *testing.T) {
	db, err := Open(t.TempDir() + "/migrations.db")
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	_, err = db.Conn().ExecContext(ctx, schemaMigrations)
	require.NoError(t, err)

	repo := NewMigrationRepository(db)
	version, err := repo.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, version)

	require.NoError(t, repo.RecordMigration(ctx, 1, "hash-1", "agent-1"))
	require.NoError(t, repo.RecordMigration(ctx, 2, "hash-2", "agent-2"))

	version, err = repo.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, version)

	migrations, err := repo.ListMigrations(ctx)
	require.NoError(t, err)
	require.Len(t, migrations, 2)
	assert.Equal(t, 1, migrations[0].Version)
	assert.Equal(t, "hash-1", migrations[0].Hash)
	assert.Equal(t, "agent-1", migrations[0].AgentVersion)
	assert.False(t, migrations[0].AppliedAtUTC.IsZero())
	assert.Equal(t, 2, migrations[1].Version)
}

func TestComputeMigrationHashDeterministicAndMigrationsMatchSQL(t *testing.T) {
	assert.Equal(t, computeMigrationHash("select 1"), computeMigrationHash("select 1"))
	assert.NotEqual(t, computeMigrationHash("select 1"), computeMigrationHash("select 2"))
	for _, migration := range Migrations {
		assert.Equal(t, computeMigrationHash(migration.SQL), migration.Hash)
	}
}

func TestApplyMigrationsAppliesSkipsAndDetectsErrors(t *testing.T) {
	db, err := Open(t.TempDir() + "/apply.db")
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	require.NoError(t, ApplyMigrations(ctx, db, "agent-1"))
	version, err := NewMigrationRepository(db).GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, version)

	require.NoError(t, ApplyMigrations(ctx, db, "agent-2"))
	migrations, err := NewMigrationRepository(db).ListMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, migrations, 1)

	original := Migrations
	t.Cleanup(func() { Migrations = original })
	Migrations = []Migration{{Version: 2, Hash: "bad", SQL: "SELECT 1;"}}
	err = ApplyMigrations(ctx, db, "agent-3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")

	Migrations = []Migration{{Version: 2, Hash: computeMigrationHash("broken sql"), SQL: "broken sql"}}
	err = ApplyMigrations(ctx, db, "agent-3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply migration")
}

func TestMigrationRepositoryClosedDBErrors(t *testing.T) {
	db, err := Open(t.TempDir() + "/closed.db")
	require.NoError(t, err)
	repo := NewMigrationRepository(db)
	require.NoError(t, db.Close())

	_, err = repo.GetVersion(context.Background())
	assert.Error(t, err)
	assert.Error(t, repo.RecordMigration(context.Background(), 1, "hash", "agent"))
	_, err = repo.ListMigrations(context.Background())
	assert.Error(t, err)
}
