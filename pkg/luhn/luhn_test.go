package luhn

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid 11-digit", "12345678903", true},
		{"valid 10-digit", "9278923470", true},
		{"valid 9-digit", "346436439", true},
		{"valid withdrawal order", "2377225624", true},

		// edge cases
		{"empty string", "", false},
		{"single zero", "0", true},
		{"single non-zero digit", "1", false},
		{"non-digit chars", "123abc", false},
		{"spaces", "1234 5678", false},

		// invalid
		{"all zeros except last", "00000000001", false},
		{"sequential digits", "1234567890", false},
		{"off by one", "12345678904", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.input); got != tt.want {
				t.Errorf("Validate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
