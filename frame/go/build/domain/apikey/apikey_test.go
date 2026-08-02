package apikey

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateKey(t *testing.T) {
	a, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if !strings.HasPrefix(a, KeyPrefix) {
		t.Errorf("key %q should have prefix %q", a, KeyPrefix)
	}
	if len(a) != 4+43 {
		t.Errorf("key length = %d, want 47 (ven_ + 43 base64url)", len(a))
	}
	b, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if a == b {
		t.Error("two generated keys should differ (randomness)")
	}
}

func TestHashKey(t *testing.T) {
	h := HashKey("ven_test")
	if len(h) != 64 {
		t.Errorf("hash length = %d, want 64 (sha256 hex)", len(h))
	}
	if h != HashKey("ven_test") {
		t.Error("HashKey should be deterministic")
	}
	if h == "ven_test" {
		t.Error("hash must not equal plaintext")
	}
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name string
		want string // 空串 = 通过
	}{
		{"", "name is required"},
		{"   ", "name is required"},
		{strings.Repeat("a", 65), "name too long (max 64)"},
		{strings.Repeat("a", 64), ""},
		{"zcode-agent", ""},
		{"  zcode-agent  ", ""}, // trim 后非空即合法
	}
	for _, c := range cases {
		if got := ValidateName(c.name); got != c.want {
			t.Errorf("ValidateName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDisplayPrefix(t *testing.T) {
	raw := "ven_ab12xyz"
	if got := DisplayPrefix(raw); got != "ven_ab12" {
		t.Errorf("DisplayPrefix(%q) = %q, want %q", raw, got, "ven_ab12")
	}
	if got := DisplayPrefix("short"); got != "short" {
		t.Errorf("DisplayPrefix(short) = %q, want %q", got, "short")
	}
}

func TestApiKeyRevoked(t *testing.T) {
	k := &ApiKey{}
	if k.Revoked() {
		t.Error("zero RevokedAt should mean not revoked")
	}
	k.RevokedAt = time.Now()
	if !k.Revoked() {
		t.Error("non-zero RevokedAt should mean revoked")
	}
}
