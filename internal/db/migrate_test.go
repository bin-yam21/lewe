package db

import "testing"

func TestMigrateURL(t *testing.T) {
	for in, want := range map[string]string{
		"postgres://u:p@h:5432/d?sslmode=disable": "pgx5://u:p@h:5432/d?sslmode=disable",
		"postgresql://h/d":                        "pgx5://h/d",
		"pgx5://h/d":                              "pgx5://h/d",
	} {
		if got := migrateURL(in); got != want {
			t.Errorf("migrateURL(%q) = %q, want %q", in, got, want)
		}
	}
}
