package core

import (
	"strings"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	src := "# 注释\nBLOG_MYSQL_DSN=\"root:pass@tcp(127.0.0.1:3306)/ven_blog?parseTime=true\"\n\nVEN_INTERNAL_TOKEN=dev-token\nKEY2='single quoted'\n  # 缩进注释\n"
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	// 条目类型序列：comment, key, blank, key, key, comment
	wantKinds := []EntryKind{EntryComment, EntryKey, EntryBlank, EntryKey, EntryKey, EntryComment}
	if len(f.Entries) != len(wantKinds) {
		t.Fatalf("条目数 = %d, want %d", len(f.Entries), len(wantKinds))
	}
	for i, k := range wantKinds {
		if f.Entries[i].Kind != k {
			t.Errorf("条目 %d kind = %v, want %v", i, f.Entries[i].Kind, k)
		}
	}

	// 键值解析（引号剥离）
	if v, ok := f.Get("BLOG_MYSQL_DSN"); !ok || v != "root:pass@tcp(127.0.0.1:3306)/ven_blog?parseTime=true" {
		t.Errorf("BLOG_MYSQL_DSN = %q, ok=%v", v, ok)
	}
	if v, ok := f.Get("VEN_INTERNAL_TOKEN"); !ok || v != "dev-token" {
		t.Errorf("VEN_INTERNAL_TOKEN = %q, ok=%v", v, ok)
	}
	if v, ok := f.Get("KEY2"); !ok || v != "single quoted" {
		t.Errorf("KEY2 = %q, ok=%v", v, ok)
	}
	if _, ok := f.Get("不存在"); ok {
		t.Error("不存在的键不应命中")
	}
}

func TestParseInvalidLines(t *testing.T) {
	if _, err := Parse([]byte("NO_EQUALS\n")); err == nil {
		t.Error("无 = 的行应报错")
	}
	if _, err := Parse([]byte("1BAD=1\n")); err == nil {
		t.Error("数字开头的键名应报错")
	}
	if _, err := Parse([]byte("A B=1\n")); err == nil {
		t.Error("含空格的键名应报错")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	src := "# head\nA=1\n\nB=\"hello world\"\n# tail\n"
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	got := string(f.Marshal())
	if got != src {
		t.Errorf("round-trip 不一致:\n got: %q\nwant: %q", got, src)
	}
}

func TestSetPreservesExistingOrderAndComments(t *testing.T) {
	f, err := Parse([]byte("# c\nA=1\nB=2\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	f.Set("B", "22") // 覆盖保留原位
	f.Set("C", "3")  // 追加
	got := string(f.Marshal())
	want := "# c\nA=1\nB=22\n\nC=3\n"
	if got != want {
		t.Errorf("Set 结果:\n got: %q\nwant: %q", got, want)
	}
}

func TestQuoteValueOnMarshal(t *testing.T) {
	f := &EnvFile{}
	f.Set("DSN", "root:p@ss#1@tcp(127.0.0.1:3306)/db?parseTime=true")
	f.Set("PLAIN", "abc")
	f.Set("EMPTY", "")
	got := string(f.Marshal())
	want := "DSN=\"root:p@ss#1@tcp(127.0.0.1:3306)/db?parseTime=true\"\nPLAIN=abc\nEMPTY=\n"
	if got != want {
		t.Errorf("Marshal:\n got: %q\nwant: %q", got, want)
	}
	// 带引号输出后应能被重新解析为原值
	f2, err := Parse([]byte(got))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f2.Get("DSN"); v != "root:p@ss#1@tcp(127.0.0.1:3306)/db?parseTime=true" {
		t.Errorf("引号值回读 = %q", v)
	}
}

func TestLoadEnvFileMissingIsEmpty(t *testing.T) {
	f, err := LoadEnvFile("C:/definitely/not/exist/.env.local")
	if err != nil {
		t.Fatalf("缺失文件应返回空文件: %v", err)
	}
	if len(f.Entries) != 0 {
		t.Errorf("空文件条目数 = %d", len(f.Entries))
	}
}

func TestKeysOrdered(t *testing.T) {
	f, _ := Parse([]byte("B=1\nA=2\nC=3\n"))
	got := strings.Join(f.Keys(), ",")
	if got != "B,A,C" {
		t.Errorf("Keys = %s", got)
	}
}
