package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnvFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultsAndValidate(t *testing.T) {
	c := &Config{Values: map[string]string{}}
	if got := c.Get(EnvToken); got != DefaultToken {
		t.Errorf("默认 token = %q, want %q", got, DefaultToken)
	}
	if c.Has(EnvToken) {
		t.Error("默认值不应算作显式配置")
	}
	if err := c.Validate(); err == nil {
		t.Error("缺少 DSN 应报错")
	}
	c.Values[EnvDSN] = "root:x@tcp(127.0.0.1:3306)/ven_blog?parseTime=true"
	if err := c.Validate(); err != nil {
		t.Errorf("DSN 已配置仍报错: %v", err)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "BLOG_MYSQL_DSN=root:x@tcp(127.0.0.1:3306)/ven_blog?parseTime=true\nVEN_INTERNAL_TOKEN=abc\n")
	c, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Get(EnvToken) != "abc" {
		t.Errorf("token = %q, want abc", c.Get(EnvToken))
	}
	if c.Get(EnvDSN) == "" {
		t.Error("DSN 应被读取")
	}
	// 文件缺失键回退默认
	c2, _ := LoadConfig(t.TempDir())
	if c2.Get(EnvToken) != DefaultToken {
		t.Errorf("空目录 token = %q", c2.Get(EnvToken))
	}
}

func TestEnvMergeFileOverridesSystem(t *testing.T) {
	t.Setenv("VEN_INTERNAL_TOKEN", "from-system")
	dir := t.TempDir()
	writeEnvFile(t, dir, "VEN_INTERNAL_TOKEN=from-file\nBLOG_MYSQL_DSN=root:x@tcp(127.0.0.1:3306)/ven_blog?parseTime=true\n")
	c, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := c.Env()
	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "VEN_INTERNAL_TOKEN=") {
			count++
			if kv != "VEN_INTERNAL_TOKEN=from-file" {
				t.Errorf("文件值应覆盖系统值: %s", kv)
			}
		}
	}
	if count != 1 {
		t.Errorf("VEN_INTERNAL_TOKEN 出现 %d 次, want 1", count)
	}
}

func TestMaskValue(t *testing.T) {
	cases := []struct {
		key, val, want string
	}{
		{"BLOG_MYSQL_DSN", "root:secret@tcp(127.0.0.1:3306)/ven_blog?parseTime=true", "root:***@tcp(127.0.0.1:3306)/ven_blog?parseTime=true"},
		{"BLOG_MYSQL_DSN", "root", "***"},
		{"VEN_INTERNAL_TOKEN", "abc", "***"},
		{"BLOG_AUTHOR_PASSWORD", "x", "***"},
		{"BLOG_LLM_API_KEY", "sk-123", "***"},
		{"BLOG_SITE_URL", "https://example.com", "https://example.com"},
		{"BLOG_AUTHOR_NAME", "author", "author"},
		{"BLOG_MYSQL_DSN", "", "(未配置)"},
	}
	for _, c := range cases {
		if got := MaskValue(c.key, c.val); got != c.want {
			t.Errorf("MaskValue(%s) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestConfigSummaryValid(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "BLOG_MYSQL_DSN=root:x@tcp(127.0.0.1:3306)/ven_blog?parseTime=true\n")
	var sb strings.Builder
	if err := ConfigSummary(dir, &sb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "校验通过") {
		t.Errorf("摘要应包含校验通过: %s", sb.String())
	}
	if strings.Contains(sb.String(), "root:x@") {
		t.Error("摘要不应泄漏 DSN 密码")
	}
}

func TestConfigSummaryMissingDSN(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "BLOG_SITE_URL=https://example.com\n")
	var sb strings.Builder
	if err := ConfigSummary(dir, &sb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "校验失败") {
		t.Errorf("摘要应提示校验失败: %s", sb.String())
	}
}

func TestIsRootAndRoot(t *testing.T) {
	dir := t.TempDir()
	if IsRoot(dir) {
		t.Error("空临时目录不应是仓库根")
	}
	if err := os.MkdirAll(filepath.Join(dir, "frame", "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "frame", "node"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsRoot(dir) {
		t.Error("含 frame/go、frame/node 的目录应是仓库根")
	}
}
