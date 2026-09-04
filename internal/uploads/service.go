// Package uploads stores item photographs.
//
// Files are written to a directory on disk and served back as static content.
// That is deliberately the simplest thing that works end to end: a listing
// without a photo is not really a listing, so the app needed a real URL to put
// in items.images long before object storage was worth configuring.
//
// The seam to replace later is Store.Save — swapping this for S3 or R2 means
// implementing that one method and leaving the handler untouched.
package uploads

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrTooLarge        = errors.New("image is too large")
)

// MaxUploadBytes caps a single image. Phone cameras produce multi-megabyte
// files, and the app already downscales before sending.
const MaxUploadBytes = 8 << 20 // 8 MiB

// extensionsByContentType is also the allowlist: anything absent is rejected.
// Executable and SVG uploads are deliberately excluded — SVG can carry script
// and would be served from our own origin.
var extensionsByContentType = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/heic": ".heic",
	"image/heif": ".heif",
}

// Store writes uploaded images to a directory.
type Store struct {
	dir string
	// publicPrefix is the URL path the directory is served under.
	publicPrefix string
}

// NewStore prepares the upload directory, creating it if necessary.
func NewStore(dir, publicPrefix string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir, publicPrefix: publicPrefix}, nil
}

// Dir returns the directory backing the store.
func (s *Store) Dir() string { return s.dir }

// Save streams an uploaded image to disk and returns its public URL path.
//
// The returned path is relative ("/uploads/<name>"), never absolute: the API is
// reached on a different host from a phone than from this machine, and a stored
// absolute URL would be wrong the moment the network changed.
func (s *Store) Save(r io.Reader, contentType string) (string, error) {
	ext, ok := extensionsByContentType[normalizeContentType(contentType)]
	if !ok {
		return "", ErrUnsupportedType
	}

	name, err := randomName()
	if err != nil {
		return "", err
	}
	name += ext

	dst, err := os.Create(filepath.Join(s.dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// LimitReader guards the disk even if the multipart parser was generous:
	// one extra byte past the cap means the client lied about the size.
	written, err := io.Copy(dst, io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		os.Remove(dst.Name())
		return "", err
	}
	if written > MaxUploadBytes {
		os.Remove(dst.Name())
		return "", ErrTooLarge
	}

	return s.publicPrefix + "/" + name, nil
}

func normalizeContentType(ct string) string {
	// "image/jpeg; charset=binary" -> "image/jpeg"
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
