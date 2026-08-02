package core

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNodeCommands(t *testing.T) {
	got := NodeCommands()
	want := [][]string{{"npm", "ci"}, {"npm", "run", "build"}}
	if len(got) != len(want) {
		t.Fatalf("命令数 = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if strings.Join(got[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("命令 %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestGoCommand(t *testing.T) {
	root := t.TempDir()
	args := GoCommand(root)
	if len(args) != 5 {
		t.Fatalf("Go 命令参数 = %v", args)
	}
	if args[0] != "go" || args[1] != "build" || args[2] != "-o" || args[4] != "." {
		t.Errorf("Go 命令结构异常: %v", args)
	}
	// 输出目标 = 仓库根/bin/<平台二进制名>
	if got, want := args[3], filepath.Join(root, "bin", BinName()); got != want {
		t.Errorf("输出路径 = %s, want %s", got, want)
	}
}

func TestBinName(t *testing.T) {
	name := BinName()
	if runtime.GOOS == "windows" {
		if name != "ven_hybird.exe" {
			t.Errorf("Windows 二进制名 = %s", name)
		}
	} else if name != "ven_hybird" {
		t.Errorf("POSIX 二进制名 = %s", name)
	}
}

func TestBinPathAndDirs(t *testing.T) {
	root := filepath.Join("C:", "repo")
	if got := filepath.Base(BinPath(root)); got != BinName() {
		t.Errorf("BinPath base = %s", got)
	}
	if NodeDir(root) != filepath.Join("C:", "repo", "frame", "node") {
		t.Errorf("NodeDir = %s", NodeDir(root))
	}
	if GoDir(root) != filepath.Join("C:", "repo", "frame", "go") {
		t.Errorf("GoDir = %s", GoDir(root))
	}
}

func TestBuildRejectsNonRoot(t *testing.T) {
	root := t.TempDir()
	err := Build(context.Background(), root, io.Discard)
	if err == nil {
		t.Fatal("非仓库根应报错")
	}
	if !strings.Contains(err.Error(), "仓库根") {
		t.Errorf("错误信息: %v", err)
	}
}

func TestBuildRejectsBrokenEnvFile(t *testing.T) {
	root := t.TempDir()
	mkRepoRoot(t, root)
	// .env.local 内容非法（无 = 的行）
	writeEnvFile(t, root, "NOT_A_KEY_VALUE\n")
	err := Build(context.Background(), root, io.Discard)
	if err == nil {
		t.Fatal("非法 .env.local 应报错")
	}
}

// mkRepoRoot 构造最小仓库根（frame/go、frame/node 目录）。
func mkRepoRoot(t *testing.T, root string) {
	t.Helper()
	for _, d := range []string{"frame/go", "frame/node"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}
