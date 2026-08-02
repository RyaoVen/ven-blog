package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BinName 返回 Go 网关二进制名（Windows 带 .exe）。
func BinName() string {
	if runtime.GOOS == "windows" {
		return "ven_hybird.exe"
	}
	return "ven_hybird"
}

// BinPath 返回 Go 网关二进制绝对路径（仓库根 bin/ 下）。
func BinPath(root string) string { return filepath.Join(root, "bin", BinName()) }

// NodeDir / GoDir 返回各子模块目录。
func NodeDir(root string) string { return filepath.Join(root, "frame", "node") }
func GoDir(root string) string   { return filepath.Join(root, "frame", "go") }

// NodeCommands 返回 Node 构建命令（npm ci + npm run build）。
func NodeCommands() [][]string {
	return [][]string{{"npm", "ci"}, {"npm", "run", "build"}}
}

// GoCommand 返回 Go 构建命令（go build -o <仓库根/bin/ven_hybird[.exe]> .）。
func GoCommand(root string) []string {
	return []string{"go", "build", "-o", BinPath(root), "."}
}

// Build 编排构建：Node（npm ci + npm run build）→ Go（go build -o bin/）。
// 输出流式打印到 w；ctx 取消时中断（Windows 下 npm 子进程可能残留，见 README）。
func Build(ctx context.Context, root string, w io.Writer) error {
	if !IsRoot(root) {
		return fmt.Errorf("未找到仓库根（缺少 frame/go、frame/node），请在仓库内运行")
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		return fmt.Errorf("读取 .env.local 失败: %w", err)
	}
	env := cfg.Env()

	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		return fmt.Errorf("创建 bin/ 失败: %w", err)
	}

	fmt.Fprintln(w, "==> 构建 Node SSR worker (frame/node) ...")
	for _, args := range NodeCommands() {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = NodeDir(root)
		cmd.Env = env
		if err := runCmd(ctx, cmd, w, "[node] "); err != nil {
			return fmt.Errorf("命令失败（%s）: %w", strings.Join(args, " "), err)
		}
	}

	fmt.Fprintf(w, "==> 构建 Go 网关 (frame/go -> %s) ...\n", BinPath(root))
	args := GoCommand(root)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = GoDir(root)
	cmd.Env = env
	if err := runCmd(ctx, cmd, w, "[go] "); err != nil {
		return fmt.Errorf("go build 失败: %w", err)
	}
	fmt.Fprintln(w, "==> 构建完成: "+BinPath(root))
	return nil
}

// runCmd 运行命令，stdout/stderr 合流并逐行加前缀输出；ctx 取消时返回 ctx.Err()。
func runCmd(ctx context.Context, cmd *exec.Cmd, w io.Writer, prefix string) error {
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return err
	}
	done := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
		for sc.Scan() {
			fmt.Fprintf(w, "%s%s\n", prefix, sc.Text())
		}
		pr.Close()
		close(done)
	}()
	err := cmd.Wait()
	pw.Close()
	<-done
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
