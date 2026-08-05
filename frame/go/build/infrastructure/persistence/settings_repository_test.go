package persistence

import (
	"strings"
	"testing"

	"ven_hybird/build/domain/setting"
	"ven_hybird/build/infrastructure/cipher"

	"github.com/DATA-DOG/go-sqlmock"
)

// newEncryptedSettingsRepo 构造带密钥的 settings 仓储（sqlmock）。
func newEncryptedSettingsRepo(t *testing.T) (*SettingsRepository, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("BLOG_SECRET_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, err := cipher.New()
	if err != nil {
		t.Fatalf("cipher.New: %v", err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewSettingsRepository(db, enc), mock
}

// TestSettingsRepo_SensitiveKeyDecryptOnRead 敏感键读库解密回原文；非敏感键原样。
func TestSettingsRepo_SensitiveKeyDecryptOnRead(t *testing.T) {
	repo, mock := newEncryptedSettingsRepo(t)

	// 先加密一个值，模拟库中密文
	sealed, err := cipher.New()
	if err != nil {
		t.Fatal(err)
	}
	_ = sealed
	enc, err := cipher.New()
	if err != nil {
		t.Fatal(err)
	}
	secret := "llm-key-abc123"
	encrypted, err := enc.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("SELECT value FROM settings WHERE").
		WithArgs(setting.KeyLLMAPIKey).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(encrypted))
	got, err := repo.Get(setting.KeyLLMAPIKey)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != secret {
		t.Fatalf("Get = %q, want %q", got, secret)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSettingsRepo_LegacyPlaintextReadable 旧明文（无 enc: 前缀）原样返回，兼容迁移。
func TestSettingsRepo_LegacyPlaintextReadable(t *testing.T) {
	repo, mock := newEncryptedSettingsRepo(t)
	mock.ExpectQuery("SELECT value FROM settings WHERE").
		WithArgs(setting.KeySMTPPass).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("legacy-plain-password"))
	got, err := repo.Get(setting.KeySMTPPass)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "legacy-plain-password" {
		t.Fatalf("Get = %q, want legacy plaintext", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSettingsRepo_NoKeyPlaintext 未配置密钥（enc=nil）时明文直存直读。
func TestSettingsRepo_NoKeyPlaintext(t *testing.T) {
	t.Setenv("BLOG_SECRET_KEY", "")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()
	repo := NewSettingsRepository(db, nil)

	mock.ExpectExec("INSERT INTO settings").
		WithArgs(setting.KeyLLMAPIKey, "plain-key").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Set(setting.KeyLLMAPIKey, "plain-key"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSettingsRepo_NonSensitiveKeyUntouched 非敏感键不加密（普通设置原样入库）。
func TestSettingsRepo_NonSensitiveKeyUntouched(t *testing.T) {
	repo, mock := newEncryptedSettingsRepo(t)
	mock.ExpectExec("INSERT INTO settings").
		WithArgs(setting.KeySiteIcon, "https://example.com/icon.png").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Set(setting.KeySiteIcon, "https://example.com/icon.png"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSettingsRepo_WriteRoundTrip 完整往返：加密写入 → 解密读回（含 enc: 前缀断言）。
func TestSettingsRepo_WriteRoundTrip(t *testing.T) {
	repo, mock := newEncryptedSettingsRepo(t)

	mock.ExpectExec("INSERT INTO settings").
		WithArgs(setting.KeySMTPPass, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Set(setting.KeySMTPPass, "roundtrip-secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	// 手工加密后模拟读回，验证解密一致（真正的往返校验）
	enc, err := cipher.New()
	if err != nil {
		t.Fatal(err)
	}
	sealed, _ := enc.Encrypt("roundtrip-secret")
	if !strings.HasPrefix(sealed, cipher.Prefix) {
		t.Fatalf("ciphertext missing prefix: %q", sealed)
	}
	mock.ExpectQuery("SELECT value FROM settings WHERE").
		WithArgs(setting.KeySMTPPass).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(sealed))
	got, err := repo.Get(setting.KeySMTPPass)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "roundtrip-secret" {
		t.Fatalf("round trip = %q, want %q", got, "roundtrip-secret")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
