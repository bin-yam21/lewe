package uploads

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{200, 100, 50, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func upload(t *testing.T, h *Handler, field string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile(field, "photo.jpg") // the client's filename and type are ignored
	fw.Write(content)
	mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/uploads", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Upload(rec, req)
	return rec
}

func TestUploadAndServe(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store)

	img := pngBytes(t)
	rec := upload(t, h, "file", img)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"/uploads/`) || !strings.Contains(body, `.png"`) {
		t.Fatalf("body = %s", body)
	}
	name := body[strings.Index(body, "/uploads/")+len("/uploads/") : strings.Index(body, `.png"`)+4]

	saved, _ := os.ReadFile(filepath.Join(dir, name))
	if !bytes.Equal(saved, img) {
		t.Error("saved file differs from upload")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /uploads/{name}", h.Serve)
	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest("GET", "/uploads/"+name, nil))
	if get.Code != 200 || get.Header().Get("Content-Type") != "image/png" || get.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("serve: %d %v", get.Code, get.Header())
	}
	for _, bad := range []string{"/uploads/..%2Fetc%2Fpasswd", "/uploads/nope.png", "/uploads/" + strings.Repeat("a", 32) + ".svg"} {
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest("GET", bad, nil))
		if r.Code != 404 {
			t.Errorf("%s: %d", bad, r.Code)
		}
	}
}

func TestUploadRejects(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	h := NewHandler(store)

	cases := []struct {
		name    string
		field   string
		content []byte
		want    int
	}{
		{"svg with script", "file", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 415},
		{"html", "file", []byte("<html><body>hi</body></html>"), 415},
		{"empty", "file", nil, 400},
		{"wrong field", "photo", pngBytes(t), 400},
		{"too large", "file", append(pngBytes(t), make([]byte, MaxBytes)...), 413},
	}
	for _, c := range cases {
		if rec := upload(t, h, c.field, c.content); rec.Code != c.want {
			t.Errorf("%s: %d, want %d (%s)", c.name, rec.Code, c.want, rec.Body)
		}
	}

	req := httptest.NewRequest("POST", "/api/v1/uploads", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Upload(rec, req)
	if rec.Code != 400 {
		t.Errorf("json body: %d", rec.Code)
	}
}
