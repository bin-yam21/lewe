// Package uploads stores item and profile photos on local disk and serves
// them back. Store is the seam to replace with object storage (S3, R2) later.
package uploads

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/yeabt/lewe/internal/response"
)

// MaxBytes caps a single image. The Mini App downsizes photos before upload,
// so this is a safety limit rather than a target.
const MaxBytes = 8 << 20 // 8 MiB

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrTooLarge        = errors.New("image is too large")
	ErrNoFile          = errors.New("no file in request")
)

// extensions is also the allowlist. SVG is deliberately absent: it can carry
// script and would be served from our own origin.
var extensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var namePattern = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|webp)$`)

// Store saves images in a directory.
type Store struct {
	dir string
}

// NewStore creates the directory if needed.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Save writes an image and returns its public path (/uploads/<name>). The
// type is detected from the file's bytes, never from the client's claim.
func (s *Store) Save(r io.Reader) (string, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		if errors.Is(err, io.EOF) {
			return "", ErrNoFile
		}
		return "", err
	}
	head = head[:n]
	ext, ok := extensions[http.DetectContentType(head)]
	if !ok {
		return "", ErrUnsupportedType
	}

	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	name := hex.EncodeToString(id) + ext

	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	written, err := io.Copy(tmp, io.LimitReader(io.MultiReader(bytes.NewReader(head), r), MaxBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	if written > MaxBytes {
		return "", ErrTooLarge
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, name)); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}

// Handler serves uploads over HTTP.
type Handler struct {
	store *Store
}

// NewHandler creates an upload handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Upload handles POST /api/v1/uploads (multipart/form-data, field "file").
// It responds with {"url": "/uploads/<name>"} to use in image_urls or avatar_url.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBytes+64<<10) // room for multipart headers
	mr, err := r.MultipartReader()
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Send the image as multipart/form-data in a field named \"file\"")
		return
	}

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			h.fail(w, err)
			return
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		url, err := h.store.Save(part)
		part.Close()
		if err != nil {
			h.fail(w, err)
			return
		}
		response.JSON(w, http.StatusCreated, map[string]string{"url": url})
		return
	}
	h.fail(w, ErrNoFile)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	switch {
	case errors.Is(err, ErrTooLarge), errors.As(err, &maxErr):
		response.Error(w, http.StatusRequestEntityTooLarge, "Images must be 8 MB or smaller")
	case errors.Is(err, ErrUnsupportedType):
		response.Error(w, http.StatusUnsupportedMediaType, "Upload a JPEG, PNG or WebP image")
	case errors.Is(err, ErrNoFile):
		response.Error(w, http.StatusBadRequest, "Send the image in a form field named \"file\"")
	default:
		response.InternalError(w, err)
	}
}

// Serve handles GET /uploads/{name}.
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !namePattern.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(h.store.dir, name))
}
