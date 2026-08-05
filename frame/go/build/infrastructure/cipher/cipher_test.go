package cipher

import (
	"strings"
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex 字符 = 32 字节

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	t.Setenv("BLOG_SECRET_KEY", testKey)
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestEncryptDecrypt_RoundTrip 加密解密往返一致；密文带前缀且非明文。
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c := newTestCipher(t)
	for _, plain := range []string{"smtp-password-123", "sk-llm-very-secret-key", "中文密码：验证码"} {
		sealed, err := c.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plain, err)
		}
		if !strings.HasPrefix(sealed, Prefix) {
			t.Fatalf("ciphertext missing prefix: %q", sealed)
		}
		if strings.Contains(sealed, plain) {
			t.Fatalf("ciphertext leaked plaintext: %q", sealed)
		}
		got, err := c.Decrypt(sealed)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if got != plain {
			t.Fatalf("round trip = %q, want %q", got, plain)
		}
	}
}

// TestEncrypt_UniqueNonce 同明文两次加密密文不同（随机 nonce）。
func TestEncrypt_UniqueNonce(t *testing.T) {
	c := newTestCipher(t)
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("two encryptions of same plaintext should differ (random nonce)")
	}
}

// TestDecrypt_LegacyPlaintext 旧明文（无前缀）原样返回，兼容迁移。
func TestDecrypt_LegacyPlaintext(t *testing.T) {
	c := newTestCipher(t)
	got, err := c.Decrypt("old-plaintext-password")
	if err != nil || got != "old-plaintext-password" {
		t.Fatalf("Decrypt(legacy) = %q, %v", got, err)
	}
}

// TestNew_MissingKey 未配置密钥返回 ErrKeyNotConfigured。
func TestNew_MissingKey(t *testing.T) {
	t.Setenv("BLOG_SECRET_KEY", "")
	if _, err := New(); err != ErrKeyNotConfigured {
		t.Fatalf("New without key = %v, want ErrKeyNotConfigured", err)
	}
}

// TestNew_BadKey 非 32 字节/非 hex 密钥拒绝。
func TestNew_BadKey(t *testing.T) {
	for _, bad := range []string{"short", "zz" + strings.Repeat("0", 30), strings.Repeat("0", 62)} {
		t.Setenv("BLOG_SECRET_KEY", bad)
		if _, err := New(); err != ErrKeyNotConfigured {
			t.Fatalf("New(%q) = %v, want ErrKeyNotConfigured", bad, err)
		}
	}
}

// TestDecrypt_Tampered 篡改密文解密失败（GCM 认证）。
func TestDecrypt_Tampered(t *testing.T) {
	c := newTestCipher(t)
	sealed, err := c.Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	// 改最后一个 base64 字符
	last := sealed[len(sealed)-1]
	alt := byte('A')
	if last == 'A' {
		alt = 'B'
	}
	tampered := sealed[:len(sealed)-1] + string(alt)
	if _, err := c.Decrypt(tampered); err == nil {
		t.Fatal("Decrypt(tampered) should fail")
	}
}
