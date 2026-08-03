package core

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess 是单测用的假进程：被 Go_WANT_HELPER_PROCESS=1 启动后长时间阻塞
// （用 time.Sleep 而非 select{}，避免 Go 运行时死锁检测触发 panic 退出）。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Println("helper alive")
	time.Sleep(24 * time.Hour)
}

// helperCmd 启动本测试二进制作为假进程（跨平台，无需真实外部程序）。
func helperCmd(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func writePidFile(t *testing.T, root string, name ProcName, pid int) {
	t.Helper()
	dir := filepath.Join(root, ".deploy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(PidPath(root, name), []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsProcessAliveWithFakeProcess(t *testing.T) {
	cmd := helperCmd(t)
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动假进程失败: %v", err)
	}
	pid := cmd.Process.Pid
	defer func() { cmd.Process.Kill(); cmd.Wait() }()

	if !IsProcessAlive(pid) {
		t.Errorf("刚启动的进程（PID %d）应判定存活", pid)
	}
	// 无效 PID
	if IsProcessAlive(0) || IsProcessAlive(-5) {
		t.Error("0/负 PID 不应存活")
	}
	if IsProcessAlive(99999999) {
		t.Error("不存在的大 PID 不应存活")
	}
	// 杀掉后判定停止
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	if IsProcessAlive(pid) {
		t.Errorf("已杀进程（PID %d）不应存活", pid)
	}
}

func TestGetStatusWithFakePidFiles(t *testing.T) {
	root := t.TempDir()
	cmd := helperCmd(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	pid := cmd.Process.Pid

	writePidFile(t, root, ProcNode, pid)
	// go 的 PID 文件指向不存在的进程
	writePidFile(t, root, ProcGo, 99999999)

	st := GetStatus(root)
	if !st.Node.Alive || st.Node.PID != pid {
		t.Errorf("node 状态 = %+v, want alive pid=%d", st.Node, pid)
	}
	if st.Go.Alive {
		t.Error("go（假 PID）不应判定存活")
	}
	if st.Node.PidFile != PidPath(root, ProcNode) {
		t.Errorf("PidFile = %s", st.Node.PidFile)
	}
	// 端口探测字段存在（本机 3306 在跑，3000/8080 空闲——不断言具体值，只验证结构）
	_ = st.Port3000
	_ = st.Port8080
	_ = st.MySQL
}

func TestGetStatusNoPidFiles(t *testing.T) {
	root := t.TempDir()
	st := GetStatus(root)
	if st.Node.Alive || st.Go.Alive {
		t.Error("无 PID 文件不应判定存活")
	}
}

func TestKillProcess(t *testing.T) {
	cmd := helperCmd(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	if err := KillProcess(pid); err != nil {
		t.Fatalf("KillProcess 失败: %v", err)
	}
	cmd.Wait()
	if IsProcessAlive(pid) {
		t.Error("KillProcess 后进程仍存活")
	}
	if err := KillProcess(0); err == nil {
		t.Error("PID 0 应报错")
	}
}

func TestStopKillsAndRemovesPidFiles(t *testing.T) {
	root := t.TempDir()
	cmd := helperCmd(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	writePidFile(t, root, ProcNode, pid)
	writePidFile(t, root, ProcGo, pid)

	var buf bytes.Buffer
	if err := Stop(root, &buf); err != nil {
		t.Fatalf("Stop 失败: %v（输出: %s）", err, buf.String())
	}
	cmd.Wait()
	if IsProcessAlive(pid) {
		t.Error("Stop 后进程仍存活")
	}
	for _, name := range []ProcName{ProcNode, ProcGo} {
		if _, err := os.Stat(PidPath(root, name)); !os.IsNotExist(err) {
			t.Errorf("PID 文件 %s 应被清理", name)
		}
	}
	if !strings.Contains(buf.String(), "已停止") {
		t.Errorf("Stop 输出: %s", buf.String())
	}

	// 重复 Stop：无 PID 文件，不报错
	if err := Stop(root, &buf); err != nil {
		t.Errorf("重复 Stop 应无错误: %v", err)
	}
}

func TestStartPreflightChecks(t *testing.T) {
	root := t.TempDir()

	// 1) 非仓库根
	if err := Start(context.Background(), root, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "仓库根") {
		t.Errorf("非仓库根应报错, got %v", err)
	}

	// 构造仓库根
	mkRepoRoot(t, root)
	writeEnvFile(t, root, "BLOG_SITE_URL=https://example.com\nVEN_INTERNAL_TOKEN=test-secret\n")
	if err := Start(context.Background(), root, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), EnvDSN) {
		t.Errorf("缺 DSN 应报错, got %v", err)
	}

	// DSN 就绪但缺构建产物
	writeEnvFile(t, root, "BLOG_MYSQL_DSN=root:x@tcp(127.0.0.1:3306)/ven_blog?parseTime=true\nVEN_INTERNAL_TOKEN=test-secret\n")
	if err := Start(context.Background(), root, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "dist/main.js") {
		t.Errorf("缺 Node 产物应报错, got %v", err)
	}

	// 补 Node 产物，缺 Go 产物
	if err := os.MkdirAll(filepath.Join(root, "frame", "node", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "frame", "node", "dist", "main.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Start(context.Background(), root, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "bin"+string(filepath.Separator)) {
		t.Errorf("缺 Go 产物应报错, got %v", err)
	}

	// 补 Go 产物，占用 :8080 → 端口占用报错。
	// 端口检查在配置/产物之后（preflight 顺序），且 Node 端口检查先于 Go 端口——
	// 本机 3000 真实被占时用例会先命中 Node 冲突，此时跳过（与 TestWaitNodeReady 同模式）。
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(BinPath(root), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if PortOpen("127.0.0.1:3000", 100*time.Millisecond) {
		t.Skip("本机 3000 被占用（Node 冲突检查先于 8080），跳过端口冲突用例")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Skipf("无法占用 8080 做测试: %v", err)
	}
	defer ln.Close()
	if err := Start(context.Background(), root, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), ":8080") {
		t.Errorf("8080 占用应报错, got %v", err)
	}
}

func TestWaitNodeReadyTimeout(t *testing.T) {
	// 端口 3000 空闲的前提下，等待必然超时
	if PortOpen("127.0.0.1:3000", 100*time.Millisecond) {
		t.Skip("本机 3000 被占用，跳过超时测试")
	}
	start := time.Now()
	var buf bytes.Buffer
	err := WaitNodeReady(context.Background(), "dev-token", 3000, 2*time.Second, &buf)
	if err == nil {
		t.Fatal("等待应超时报错")
	}
	if elapsed := time.Since(start); elapsed < 2*time.Second {
		t.Errorf("超时过快: %v", elapsed)
	}
	if !strings.Contains(buf.String(), "等待") {
		t.Errorf("应输出等待提示: %s", buf.String())
	}
}

func TestWaitNodeReadyCanceled(t *testing.T) {
	if PortOpen("127.0.0.1:3000", 100*time.Millisecond) {
		t.Skip("本机 3000 被占用，跳过取消测试")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()
	err := WaitNodeReady(ctx, "dev-token", 3000, 30*time.Second, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Errorf("ctx 取消应返回取消错误, got %v", err)
	}
}

// freePort 申请一个端口后立即释放（返回的端口大概率空闲，用于超时类测试）。
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("申请空闲端口失败: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestWaitGoReadyTimeout(t *testing.T) {
	port := freePort(t)
	start := time.Now()
	var buf bytes.Buffer
	err := WaitGoReady(context.Background(), port, 2*time.Second, &buf)
	if err == nil {
		t.Fatal("等待应超时报错")
	}
	if elapsed := time.Since(start); elapsed < 2*time.Second {
		t.Errorf("超时过快: %v", elapsed)
	}
	if !strings.Contains(err.Error(), "/api/site") {
		t.Errorf("错误应提及 /api/site: %v", err)
	}
}

func TestWaitGoReadyAccepts4xx(t *testing.T) {
	// 网关进程活着但返回 4xx（配置/路由异常）也算"活着"——就绪语义不含健康检查
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()
	port, err := strconv.Atoi(strings.TrimPrefix(srv.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WaitGoReady(context.Background(), port, 2*time.Second, &bytes.Buffer{}); err != nil {
		t.Errorf("4xx 响应应判定就绪, got %v", err)
	}
}

func TestWaitGoReadyCanceled(t *testing.T) {
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()
	err := WaitGoReady(ctx, port, 30*time.Second, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Errorf("ctx 取消应返回取消错误, got %v", err)
	}
}
