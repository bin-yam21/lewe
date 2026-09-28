package validator

import "testing"

func TestValidators(t *testing.T) {
	errs := make(Errors)
	ValidateEmail(errs, "ok", "a.b+c@example.co")
	ValidateEmail(errs, "empty", "  ")
	ValidateEmail(errs, "bad", "not-an-email")
	ValidateRequired(errs, "req", " ")
	ValidateMinLength(errs, "min", "abc", 4)
	ValidateMaxLength(errs, "max", "abcde", 4)
	ValidateOneOf(errs, "oneof_ok", "b", []string{"a", "b"})
	ValidateOneOf(errs, "oneof_bad", "z", []string{"a", "b"})
	ValidateRange(errs, "range_ok", 3, 1, 5)
	ValidateRange(errs, "range_bad", 0, 1, 5)

	want := map[string]string{
		"empty":     "is required",
		"bad":       "is not a valid email address",
		"req":       "is required",
		"min":       "must be at least 4 characters",
		"max":       "must be at most 4 characters",
		"oneof_bad": "must be one of: a, b",
		"range_bad": "must be between 1 and 5",
	}
	if len(errs) != len(want) {
		t.Errorf("got %d errors, want %d: %v", len(errs), len(want), errs)
	}
	for field, msg := range want {
		if errs[field] != msg {
			t.Errorf("%s: got %q, want %q", field, errs[field], msg)
		}
	}
	if !errs.HasErrors() || errs.Error() == "" {
		t.Error("expected HasErrors and a non-empty message")
	}
}
