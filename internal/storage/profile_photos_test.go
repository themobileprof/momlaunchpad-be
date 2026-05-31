package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfilePhotoStoreSaveAndDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewProfilePhotoStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	path, err := store.Save("user-1", jpegHeader)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !IsManagedProfilePhotoPath(path) {
		t.Fatalf("unexpected path: %s", path)
	}

	rel := filepath.Join(dir, "profile-photos", "user-1")
	entries, err := os.ReadDir(rel)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one saved file, got err=%v len=%d", err, len(entries))
	}

	if err := store.DeleteByPublicPath(path); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
