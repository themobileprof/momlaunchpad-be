package storage

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxProfilePhotoBytes = 5 << 20 // 5MB

var profilePhotoExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ProfilePhotoStore saves profile photos on local disk under rootDir.
type ProfilePhotoStore struct {
	rootDir string
}

// NewProfilePhotoStore creates the upload directory tree.
func NewProfilePhotoStore(rootDir string) (*ProfilePhotoStore, error) {
	if rootDir == "" {
		rootDir = "./uploads"
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "profile-photos"), 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &ProfilePhotoStore{rootDir: rootDir}, nil
}

// Root returns the filesystem root used for uploads.
func (s *ProfilePhotoStore) Root() string {
	return s.rootDir
}

// Save stores image bytes for a user and returns a public URL path (e.g. /uploads/profile-photos/...).
func (s *ProfilePhotoStore) Save(userID string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty file")
	}
	if len(data) > MaxProfilePhotoBytes {
		return "", fmt.Errorf("file too large")
	}

	contentType := http.DetectContentType(data)
	ext, ok := profilePhotoExtensions[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported image type")
	}

	userDir := filepath.Join(s.rootDir, "profile-photos", userID)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return "", fmt.Errorf("create user dir: %w", err)
	}

	if err := s.removeUserFiles(userDir); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	fullPath := filepath.Join(userDir, filename)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/uploads/profile-photos/" + userID + "/" + filename, nil
}

// DeleteByPublicPath removes a previously stored photo if it lives under this store.
func (s *ProfilePhotoStore) DeleteByPublicPath(publicPath string) error {
	path := managedUploadPath(publicPath)
	if path == "" {
		return nil
	}
	rel := strings.TrimPrefix(path, "/uploads/")
	fullPath := filepath.Join(s.rootDir, rel)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	userDir := filepath.Dir(fullPath)
	_ = os.Remove(userDir)
	return nil
}

// IsManagedProfilePhotoPath reports whether a URL/path points at this server's uploads.
func IsManagedProfilePhotoPath(value string) bool {
	return strings.Contains(value, "/uploads/profile-photos/")
}

func managedUploadPath(value string) string {
	idx := strings.Index(value, "/uploads/profile-photos/")
	if idx < 0 {
		return ""
	}
	return value[idx:]
}

func (s *ProfilePhotoStore) removeUserFiles(userDir string) error {
	entries, err := os.ReadDir(userDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(userDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
