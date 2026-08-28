package resolver

import (
	"context"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/rs/zerolog"
)

func TestBadgerResolver_StoreAndLoadEntry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "badger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	opts := badger.DefaultOptions(tempDir).WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()
	res := NewBadgerResolver(&logger, db)

	data := &sourcetype.SourceData{
		Signature: "test-sig-1",
		Source:    "test-source",
	}

	ctx := context.Background()
	if err := res.StoreEntry(ctx, "test-sig-1", data); err != nil {
		t.Fatalf("StoreEntry failed: %v", err)
	}

	entries, err := res.LoadEntries(ctx, []string{"test-sig-1"})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Signature != "test-sig-1" || entries[0].Source != "test-source" {
		t.Errorf("unexpected entry loaded: %+v", entries[0])
	}
}
