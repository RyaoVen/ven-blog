package config

import "testing"

// VEN_COOKIE_SECURE 默认 true：鉴权 cookie 仅 HTTPS 发送（安全默认）。
func TestLoad_CookieSecureDefaultTrue(t *testing.T) {
	t.Setenv("VEN_COOKIE_SECURE", "") // 显式置空，避免本机环境变量干扰断言
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("VEN_COOKIE_SECURE 默认应为 true")
	}
}

// VEN_COOKIE_SECURE=false：本地 http 开发可关掉 Secure（否则 http 不传 cookie 登不上）。
func TestLoad_CookieSecureEnvDisable(t *testing.T) {
	t.Setenv("VEN_COOKIE_SECURE", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.CookieSecure {
		t.Error("VEN_COOKIE_SECURE=false 时应关闭 Secure")
	}
}
