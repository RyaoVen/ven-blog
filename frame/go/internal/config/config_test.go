package config

import (
	"os"
	"testing"
)

// 测试用强令牌：Load 拒绝空值/默认值，所有用例需显式配置。
const testInternalToken = "test-internal-token-9f2c"

// VEN_COOKIE_SECURE 默认 true：鉴权 cookie 仅 HTTPS 发送（安全默认）。
func TestLoad_CookieSecureDefaultTrue(t *testing.T) {
	t.Setenv("VEN_INTERNAL_TOKEN", testInternalToken)
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
	t.Setenv("VEN_INTERNAL_TOKEN", testInternalToken)
	t.Setenv("VEN_COOKIE_SECURE", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.CookieSecure {
		t.Error("VEN_COOKIE_SECURE=false 时应关闭 Secure")
	}
}

// 内部令牌为安全关键配置：未显式设置（回退 development-token）时拒绝启动。
func TestLoad_InternalTokenDefaultRejected(t *testing.T) {
	old, had := os.LookupEnv("VEN_INTERNAL_TOKEN")
	os.Unsetenv("VEN_INTERNAL_TOKEN")
	t.Cleanup(func() {
		if had {
			os.Setenv("VEN_INTERNAL_TOKEN", old)
		} else {
			os.Unsetenv("VEN_INTERNAL_TOKEN")
		}
	})
	if _, err := Load(); err == nil {
		t.Fatal("默认 development-token 应拒绝启动")
	}
}

// 显式置空同样拒绝启动：内部通道不允许无令牌运行（fail-open 已移除）。
func TestLoad_InternalTokenEmptyRejected(t *testing.T) {
	t.Setenv("VEN_INTERNAL_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("空令牌应拒绝启动")
	}
}

// 显式配置强令牌后正常启动，令牌按原值生效。
func TestLoad_InternalTokenCustomAccepted(t *testing.T) {
	t.Setenv("VEN_INTERNAL_TOKEN", testInternalToken)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.InternalToken != testInternalToken {
		t.Errorf("InternalToken = %q, want %q", cfg.InternalToken, testInternalToken)
	}
}
