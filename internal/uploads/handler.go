package uploads

import (
	"errors"
	"net/http"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/response"
)

// Handler serves the image upload endpoint.
type Handler struct {
	store *Store
}

// NewHandler creates a new uploads handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// UploadResponse is returned for a successfully stored image.
type UploadResponse struct {
	// URL is a path relative to the API root, e.g. "/uploads/ab12….jpg".
	// The client joins it to whichever host it reached the API on.
	URL string `json:"url"`
}

// Upload handles POST /api/v1/uploads (multipart/form-data, field "file").
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if _, err := middleware.UserIDFromContext(r.Context()); err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Reject oversized bodies before reading them into memory or onto disk.
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+1024)

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Expected a multipart form with a 'file' field")
		return
	}
	defer file.Close()

	if header.Size > MaxUploadBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, "Image must be 8MB or smaller")
		return
	}

	contentType := header.Header.Get("Content-Type")

	url, err := h.store.Save(file, contentType)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedType):
			response.Error(w, http.StatusUnsupportedMediaType, "Only JPEG, PNG, WebP and HEIC images are supported")
		case errors.Is(err, ErrTooLarge):
			response.Error(w, http.StatusRequestEntityTooLarge, "Image must be 8MB or smaller")
		default:
			response.Error(w, http.StatusInternalServerError, "Could not store the image")
		}
		return
	}

	response.JSON(w, http.StatusCreated, UploadResponse{URL: url})
}
