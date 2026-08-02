package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ProcName 进程名：node worker / go gateway。
type ProcName string

const (
	ProcNode ProcName = "node"
	ProcGo   ProcName = "go"
)

// PidPath 返回 PID 文件路径（仓库根 .deploy/<name>.pid）。
func PidPath(root string, name ProcName) string {
	return filepath.Join(root, ".deploy", string(name)+".pid")
}

// LogPath 返回日志文件路径（仓库根 logs/<name>.log）。
func LogPath(root string, name ProcName) string {
	return filepath.Join(root, "logs", string(name)+".log")
}

// ProcState 单个进程的状态。
type ProcState struct {
	Name    ProcName
	PID     int
	Alive   bool
	PidFile string
	Log     string
}

// Status 整体运行状态（进程 + 端口）。
type Status struct {
	Node     ProcState
	Go       ProcState
	Port3000 bool
	Port8080 bool
	MySQL    bool
}

// GetStatus 汇总当前状态：PID 文件 + 存活判定 + 端口探测。
func GetStatus(root string) Status {
	return Status{
		Node:     readProcState(root, ProcNode),
		Go:       readProcState(root, ProcGo),
		Port3000: PortOpen("127.0.0.1:3000", 300*time.Millisecond),
		Port8080: PortOpen("127.0.0.1:8080", 300*time.Millisecond),
		MySQL:    PortOpen("127.0.0.1:3306", 300*time.Millisecond),
	}
}

func readProcState(root string, name ProcName) ProcState {
	p := ProcState{Name: name, PidFile: PidPath(root, name), Log: LogPath(root, name)}
	data, err := os.ReadFile(p.PidFile)
	if err != nil {
		return p
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return p
	}
	p.PID = pid
	p.Alive = IsProcessAlive(pid)
	return p
}

var tasklistRe = regexp.MustCompile(`(?m)^\S+\s+(\d+)\s`)

// IsProcessAlive 判断 PID 对应进程是否存活。
// Windows 用 tasklist 查询（按 PID 精确匹配，避免本地化输出差异）；
// POSIX 用 Signal(0) 探测（等价 kill(pid, 0)）。
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
		if err != nil {
			return false
		}
		want := strconv.Itoa(pid)
		for _, m := range tasklistRe.FindAllSubmatch(out, -1) {
			if string(m[1]) == want {
				return true
			}
		}
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

// KillProcess 强杀进程：Windows 先 TerminateProcess，失败时 taskkill /T /F 兜底；
// POSIX 用 os.Process.Kill（SIGKILL）。
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效 PID: %d", pid)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("查找 PID %d 失败: %w", pid, err)
	}
	if runtime.GOOS == "windows" {
		if kerr := p.Kill(); kerr == nil {
			return nil
		}
		out, err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).CombinedOutput()
		if err == nil {
			return nil
		}
		return fmt.Errorf("强杀 PID %d 失败: %v: %s", pid, err, strings.TrimSpace(string(out)))
	}
	return p.Kill()
}

// WaitNodeReady 轮询 GET /pages（带 X-Ven-Internal-Token 头），timeout 内返回 200 即就绪。
// ctx 取消时立即返回 ctx.Err()（用于 Ctrl+C 中断启动流程）。
func WaitNodeReady(ctx context.Context, token string, timeout time.Duration, w io.Writer) error {
	client := &http.Client{Timeout: 2 * time.Second}
	const url = "http://127.0.0.1:3000/pages"
	deadline := time.Now().Add(timeout)
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err == nil {
			req.Header.Set("X-Ven-Internal-Token", token)
			if resp, rerr := client.Do(req); rerr == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Node worker 未在 %s 内就绪（GET %s 未返回 200）", timeout, url)
		}
		fmt.Fprintf(w, "  等待 Node worker 就绪（GET /pages，%s 超时）...\n", time.Until(deadline).Round(time.Second))
		time.Sleep(time.Second)
	}
}

// Start 编排启动：Node 先起 → 等 /pages 就绪（30s）→ Go 后起。
// 环境变量从 .env.local 注入；stdout/stderr 重定向 logs/；PID 写 .deploy/。
// ctx 取消时中止流程并清理已启动的进程。
func Start(ctx context.Context, root string, w io.Writer) error {
	if !IsRoot(root) {
		return fmt.Errorf("未找到仓库根（缺少 frame/go、frame/node），请在仓库内运行")
	}

	st := GetStatus(root)
	if st.Node.Alive || st.Port3000 {
		return fmt.Errorf("Node worker 已在运行（PID %d 或 :3000 被占用），请先 stop", st.Node.PID)
	}
	if st.Go.Alive || st.Port8080 {
		return fmt.Errorf("Go 网关已在运行（PID %d 或 :8080 被占用），请先 stop", st.Go.PID)
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		return fmt.Errorf("读取 .env.local 失败: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(NodeDir(root), "dist", "main.js")); err != nil {
		return fmt.Errorf("缺少 Node 构建产物 frame/node/dist/main.js，请先 build")
	}
	if _, err := os.Stat(BinPath(root)); err != nil {
		return fmt.Errorf("缺少 Go 构建产物 %s，请先 build", BinPath(root))
	}

	for _, d := range []string{filepath.Join(root, ".deploy"), filepath.Join(root, "logs")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("创建 %s 失败: %w", d, err)
		}
	}

	// 1) Node worker
	fmt.Fprintln(w, "==> 启动 Node SSR worker（node dist/main.js，:3000）...")
	nodeLog, err := openLog(LogPath(root, ProcNode))
	if err != nil {
		return err
	}
	fmt.Fprintf(nodeLog, "\n[%s] ====== deploy start ======\n", time.Now().Format("2006-01-02 15:04:05"))
	nodeCmd := exec.Command("node", "dist/main.js")
	nodeCmd.Dir = NodeDir(root)
	nodeCmd.Env = cfg.Env()
	nodeCmd.Stdout = nodeLog
	nodeCmd.Stderr = nodeLog
	if err := nodeCmd.Start(); err != nil {
		nodeLog.Close()
		return fmt.Errorf("启动 Node 失败: %w", err)
	}
	nodePid := nodeCmd.Process.Pid
	if err := os.WriteFile(PidPath(root, ProcNode), []byte(strconv.Itoa(nodePid)+"\n"), 0o644); err != nil {
		_ = KillProcess(nodePid)
		nodeLog.Close()
		return fmt.Errorf("写 PID 文件失败: %w", err)
	}
	fmt.Fprintf(w, "  Node worker PID=%d\n", nodePid)

	// 2) 等 Node 就绪
	if err := WaitNodeReady(ctx, cfg.InternalToken(), 30*time.Second, w); err != nil {
		_ = KillProcess(nodePid)
		fmt.Fprintln(w, "  已清理未就绪的 Node 进程")
		return err
	}

	// 3) Go 网关
	fmt.Fprintln(w, "==> 启动 Go 网关（bin/ven_hybird，:8080）...")
	goLog, err := openLog(LogPath(root, ProcGo))
	if err != nil {
		return err
	}
	fmt.Fprintf(goLog, "\n[%s] ====== deploy start ======\n", time.Now().Format("2006-01-02 15:04:05"))
	goCmd := exec.Command(BinPath(root))
	goCmd.Dir = GoDir(root)
	goCmd.Env = cfg.Env()
	goCmd.Stdout = goLog
	goCmd.Stderr = goLog
	if err := goCmd.Start(); err != nil {
		goLog.Close()
		return fmt.Errorf("启动 Go 网关失败: %w", err)
	}
	goPid := goCmd.Process.Pid
	if err := os.WriteFile(PidPath(root, ProcGo), []byte(strconv.Itoa(goPid)+"\n"), 0o644); err != nil {
		_ = KillProcess(goPid)
		goLog.Close()
		return fmt.Errorf("写 PID 文件失败: %w", err)
	}
	fmt.Fprintf(w, "  Go 网关 PID=%d\n", goPid)
	fmt.Fprintln(w, "==> 启动完成。日志: logs/node.log、logs/go.log；停止: deploy stop")
	return nil
}

func openLog(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开日志 %s 失败: %w", path, err)
	}
	return f, nil
}

// Stop 按 PID 文件强杀进程（Go 先、Node 后），并清理 PID 文件。
func Stop(root string, w io.Writer) error {
	var failed []string
	for _, name := range []ProcName{ProcGo, ProcNode} {
		st := readProcState(root, name)
		if st.PID == 0 {
			fmt.Fprintf(w, "[%s] 无 PID 文件（%s），跳过\n", name, st.PidFile)
			continue
		}
		if !st.Alive {
			fmt.Fprintf(w, "[%s] 进程已不在运行（PID %d 文件过期）\n", name, st.PID)
		} else if err := KillProcess(st.PID); err != nil {
			fmt.Fprintf(w, "[%s] 强杀 PID %d 失败: %v\n", name, st.PID, err)
			failed = append(failed, string(name))
			continue
		} else {
			fmt.Fprintf(w, "[%s] 已停止（PID %d）\n", name, st.PID)
		}
		if err := os.Remove(st.PidFile); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(w, "[%s] 清理 PID 文件失败: %v\n", name, err)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("停止失败: %s", strings.Join(failed, "、"))
	}
	return nil
}

// Restart 停止后重新启动。
func Restart(ctx context.Context, root string, w io.Writer) error {
	if err := Stop(root, w); err != nil {
		return err
	}
	return Start(ctx, root, w)
}
