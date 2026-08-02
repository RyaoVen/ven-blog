package guestbook

import "testing"

func TestValidateRejectedReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		wantOK bool
	}{
		{"empty", "", false},
		{"whitespace only", "   \t\n ", false},
		{"single char", "a", true},
		{"exactly 200 chars", string(make([]byte, 200)), true},
		{"201 chars", string(make([]byte, 201)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ValidateRejectedReason(tt.reason)
			if (msg == "") != tt.wantOK {
				t.Fatalf("ValidateRejectedReason(%q) = %q, wantOK=%v", tt.reason, msg, tt.wantOK)
			}
		})
	}
}
