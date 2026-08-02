package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	lines := []string{
		"line-0",
		"line-1",
		"line-2",
		"line-3",
		"line-4",
		"line-5",
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Tail(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"line-3", "line-4", "line-5"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Tail(3) = %v, want %v", got, want)
	}

	// 超出行数取全部
	got, err = Tail(path, 100)
	if err != nil || len(got) != 6 {
		t.Errorf("Tail(100) = %d 行, err=%v", len(got), err)
	}

	// 文件末尾无换行
	if err := os.WriteFile(path, []byte("a\nb\nc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Tail(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "|") != "a|b|c" {
		t.Errorf("无尾换行 Tail = %v", got)
	}

	// Windows \r\n
	if err := os.WriteFile(path, []byte("x\r\ny\r\nz\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Tail(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "|") != "x|y|z" {
		t.Errorf("CRLF Tail = %v", got)
	}

	// 空文件
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Tail(path, 10)
	if err != nil || len(got) != 0 {
		t.Errorf("空文件 Tail = %v, err=%v", got, err)
	}
}

func TestTailMissingFile(t *testing.T) {
	_, err := Tail(filepath.Join(t.TempDir(), "nope.log"), 10)
	if err == nil {
		t.Fatal("缺失文件应报错")
	}
	if !strings.Contains(err.Error(), "未启动") {
		t.Errorf("错误信息: %v", err)
	}
}
