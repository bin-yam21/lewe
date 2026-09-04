package validator

import "testing"

func TestValidateRequired(t *testing.T) {
	errs := make(Errors)
	ValidateRequired(errs, "title", "")

	if !errs.HasErrors() {
		t.Fatalf("expected an error for empty title, got none")
	}
	if msg := errs["title"]; msg != "is required" {
		t.Errorf("errs[%q] = %q, want %q", "title", msg, "is required")
	}
}

func TestValidateEmail(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "a@b.com", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "missing at sign", input: "a-b.com", wantErr: true},
		{name: "missing tld", input: "a@b", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := make(Errors)
			ValidateEmail(errs, "email", tc.input)

			got := errs.HasErrors()
			if got != tc.wantErr {
				t.Errorf("ValidateEmail(%q): HasErrors() = %v, want %v (errs=%v)", tc.input, got, tc.wantErr, errs)
			}
		})
	}
}
