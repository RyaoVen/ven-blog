package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DetectResult 是环境检测的结构化结果。
type DetectResult struct {
	GoVersion   string // go version 输出
	NodeVersion string // node --version 输出
	NpmVersion  string // npm --version 输出
	MySQL3306   bool   // 3306 TCP 可达
	Port3000    bool   // :3000 被占用（Node worker）
	Port8080    bool   // :8080 被占用（Go 网关）
	EnvFile     bool   // .env.local 存在
	DSNSet      bool   // BLOG_MYSQL_DSN 已配置
}

// Detect 在仓库根（Root()）下执行环境检测。
func Detect() DetectResult {
	return DetectWithRoot(Root())
}

// DetectWithRoot 执行环境检测（各版本命令 5s 超时）。
// 端口从 .env.local 读取（VEN_NODE_PORT / VEN_LISTEN_ADDR），读取失败回退默认值。
func DetectWithRoot(root string) DetectResult {
	r := DetectResult{}
	r.GoVersion, _ = VersionOf("go", "version")
	r.NodeVersion, _ = VersionOf("node", "--version")
	r.NpmVersion, _ = VersionOf("npm", "--version")
	r.MySQL3306 = PortOpen("127.0.0.1:3306", 500*time.Millisecond)
	nodePort, goPort := DefaultNodePort, DefaultGoPort
	if c, err := LoadConfig(root); err == nil {
		nodePort, goPort = c.NodePort(), c.GoPort()
	}
	r.Port3000 = PortOpen(addrOf(nodePort), 500*time.Millisecond)
	r.Port8080 = PortOpen(addrOf(goPort), 500*time.Millisecond)
	if root != "" {
		if _, err := os.Stat(filepath.Join(root, ".env.local")); err == nil {
			r.EnvFile = true
			if c, err := LoadConfig(root); err == nil {
				r.DSNSet = c.Get(EnvDSN) != ""
			}
		}
	}
	return r
}

// PortOpen 探测 TCP 端口是否可连接（true = 有服务在监听）。
func PortOpen(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// VersionOf 执行版本命令并返回去首尾空白的 stdout；失败返回 ("", false)。
func VersionOf(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// Check 检测环境并输出表格到 w；核心项（go/node/npm/MySQL）缺失时返回错误。
func Check(root string, w io.Writer) error {
	r := DetectWithRoot(root)
	nodePort, goPort := DefaultNodePort, DefaultGoPort
	if c, err := LoadConfig(root); err == nil {
		nodePort, goPort = c.NodePort(), c.GoPort()
	}
	ver := func(v string) string {
		if v == "" {
			return "未找到"
		}
		return v
	}
	mark := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "✗"
	}
	fmt.Fprintf(w, "[GO]      %-44s %s\n", ver(r.GoVersion), mark(r.GoVersion != ""))
	fmt.Fprintf(w, "[NODE]    %-44s %s\n", ver(r.NodeVersion), mark(r.NodeVersion != ""))
	fmt.Fprintf(w, "[NPM]     %-44s %s\n", ver(r.NpmVersion), mark(r.NpmVersion != ""))
	fmt.Fprintf(w, "[MySQL]   127.0.0.1:3306 可达              %s\n", mark(r.MySQL3306))
	port3000 := fmt.Sprintf("空闲（Node worker 就绪位 :%d）", nodePort)
	if r.Port3000 {
		port3000 = fmt.Sprintf("占用（疑似已在运行 :%d）", nodePort)
	}
	fmt.Fprintf(w, "[端口]    :%-3d %-34s %s\n", nodePort, port3000, mark(!r.Port3000))
	port8080 := fmt.Sprintf("空闲（Go 网关就绪位 :%d）", goPort)
	if r.Port8080 {
		port8080 = fmt.Sprintf("占用（疑似已在运行 :%d）", goPort)
	}
	fmt.Fprintf(w, "[端口]    :%-3d %-34s %s\n", goPort, port8080, mark(!r.Port8080))
	env := "不存在（运行 config 生成）"
	if r.EnvFile {
		env = "存在"
	}
	if r.DSNSet {
		env += "，BLOG_MYSQL_DSN 已配置"
	} else if r.EnvFile {
		env += "，BLOG_MYSQL_DSN 未配置！"
	}
	fmt.Fprintf(w, "[配置]    .env.local %-33s %s\n", env, mark(r.EnvFile && r.DSNSet))

	var problems []string
	if r.GoVersion == "" {
		problems = append(problems, "Go 未安装")
	}
	if r.NodeVersion == "" {
		problems = append(problems, "Node 未安装")
	}
	if r.NpmVersion == "" {
		problems = append(problems, "npm 未安装")
	}
	if !r.MySQL3306 {
		problems = append(problems, "MySQL (3306) 不可达")
	}
	if len(problems) > 0 {
		return fmt.Errorf("环境检查未通过：%s", strings.Join(problems, "；"))
	}
	return nil
}
