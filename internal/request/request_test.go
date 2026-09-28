package request

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPagination(t *testing.T) {
	cases := map[string]Page{
		"/":                      {Limit: DefaultLimit},
		"/?limit=5&offset=10":    {Limit: 5, Offset: 10},
		"/?limit=1000":           {Limit: MaxLimit},
		"/?limit=-1&offset=-5":   {Limit: DefaultLimit},
		"/?limit=abc&offset=xyz": {Limit: DefaultLimit},
	}
	for url, want := range cases {
		if got := Pagination(httptest.NewRequest("GET", url, nil)); got != want {
			t.Errorf("%s: got %+v, want %+v", url, got, want)
		}
	}
}

func TestDecodeJSON(t *testing.T) {
	var dst struct{ A int }
	for body, wantErr := range map[string]bool{
		`{"a": 1}`:          false,
		`{"a": 1} {"a": 2}`: true,
		`not json`:          true,
		``:                  true,
	} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		err := DecodeJSON(httptest.NewRecorder(), r, &dst)
		if (err != nil) != wantErr {
			t.Errorf("%q: err = %v, wantErr %v", body, err, wantErr)
		}
	}
}

func TestParseUUID(t *testing.T) {
	if _, err := ParseUUID("7f1c2a8e-3b1d-4c5e-9f00-112233445566"); err != nil {
		t.Error(err)
	}
	if _, err := ParseUUID("nope"); err != ErrInvalidID {
		t.Errorf("err = %v", err)
	}
}
