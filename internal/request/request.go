package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// MaxBodyBytes caps JSON request bodies to protect the server from oversized payloads.
const MaxBodyBytes = 1 << 20 // 1 MiB

var (
	ErrInvalidBody = errors.New("invalid request body")
	ErrInvalidID   = errors.New("invalid id")
)

// DecodeJSON decodes a size-limited JSON request body into dst.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return ErrInvalidBody
	}
	// Reject trailing data such as a second JSON document.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrInvalidBody
	}
	return nil
}

// ParseUUID parses a UUID string into a pgtype.UUID.
func ParseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, ErrInvalidID
	}
	return id, nil
}

// PathUUID parses the named path wildcard (e.g. {id}) as a UUID.
func PathUUID(r *http.Request, name string) (pgtype.UUID, error) {
	return ParseUUID(r.PathValue(name))
}

// Page holds limit/offset pagination parameters.
type Page struct {
	Limit  int
	Offset int
}

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Pagination reads ?limit= and ?offset= from the query string, clamping to sane bounds.
func Pagination(r *http.Request) Page {
	p := Page{Limit: DefaultLimit}
	q := r.URL.Query()
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		p.Limit = min(v, MaxLimit)
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v > 0 {
		p.Offset = v
	}
	return p
}
