package dual

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestExecInTransaction_RecoversPanicAndRollsBack(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = ExecInTransaction(context.Background(), db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO items (id, value) VALUES (1, 'unsafe')`); err != nil {
			t.Fatalf("insert item: %v", err)
		}
		panic("boom")
	})
	if err == nil {
		t.Fatal("expected recovered panic error")
	}
	if !strings.Contains(err.Error(), "transaction panic recovered") {
		t.Fatalf("expected recovered panic error, got %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback after panic, got %d rows", count)
	}
}
