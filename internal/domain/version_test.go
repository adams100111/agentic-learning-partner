package domain

import "testing"

func TestSatisfiesVersionRange(t *testing.T) {
	tests := []struct {
		version string
		rng     string
		want    bool
	}{
		{"1.2.0", ">=1.2 <2.0", true},
		{"1.2.0", ">=1.3 <2.0", false},
		{"0.1.0", ">=0.1 <0.2", true},
		{"2.0.0", "<2.0", false},
	}

	for _, tt := range tests {
		got, err := Satisfies(tt.version, tt.rng)
		if err != nil {
			t.Fatalf("Satisfies(%q, %q): %v", tt.version, tt.rng, err)
		}
		if got != tt.want {
			t.Fatalf("Satisfies(%q, %q) = %v, want %v", tt.version, tt.rng, got, tt.want)
		}
	}
}
