package firestore

import (
	"context"
	"os"
	"testing"
)

func TestNewFirestoreStorage(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing project ID",
			cfg: Config{
				CollectionID: "test-collection",
			},
			wantErr: true,
		},
		{
			name: "missing collection ID",
			cfg: Config{
				ProjectID: "test-project",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := NewFirestoreStorage(ctx, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFirestoreStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestFirestoreStorage_GetSet runs against the Firestore Local Emulator.
// Start the emulator first with: gcloud emulators firestore start
// and set FIRESTORE_EMULATOR_HOST accordingly (the emulator prints this on startup).
func TestFirestoreStorage_GetSet(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Skipping Firestore integration test - FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	storage, err := NewFirestoreStorage(ctx, Config{
		ProjectID:    "test-project",
		CollectionID: "test_changelogs",
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test Get on non-existent changelog
	changelog, found, err := storage.Get(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() found = true, want false for non-existent changelog")
	}
	if changelog != "" {
		t.Errorf("Get() changelog = %q, want empty string", changelog)
	}

	// Test Set
	testChangelog := "Test changelog content"
	err = storage.Set(ctx, "1.0.0", "2.0.0", testChangelog)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Test Get after Set
	changelog, found, err = storage.Get(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() found = false, want true after Set")
	}
	if changelog != testChangelog {
		t.Errorf("Get() changelog = %q, want %q", changelog, testChangelog)
	}

	// Test List
	entries, err := storage.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	found = false
	for _, entry := range entries {
		if entry.FromVersion == "1.0.0" && entry.ToVersion == "2.0.0" {
			found = true
			if entry.Changelog != testChangelog {
				t.Errorf("List() changelog = %q, want %q", entry.Changelog, testChangelog)
			}
		}
	}
	if !found {
		t.Error("List() did not return the entry stored by Set()")
	}
}
