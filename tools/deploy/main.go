// Command ven-blog 部署工具：bubbletea TUI + 子命令
// （check/config/build/start/stop/restart/status/logs），跨平台（Windows/Linux）。
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ven_hybird/tools/deploy/core"
	"ven_hybird/tools/deploy/tui"
)

const usage = `ven-blog 跨平台部署工具（tools/deploy，issue #112）

用法（在仓库根或 tools/deploy/ 目录运行，自动定位仓库根）:
  deploy                启动 TUI 主界面
  deploy check          环境检测（go/node/npm/MySQL 3306/端口 3000 8080）
  deploy config         查看/生成 .env.local（无文件时交互问答）
  deploy config --set KEY=VALUE [--set K2=V2]
                        追加/覆盖配置项（保留已有键，校验 BLOG_MYSQL_DSN）
  deploy build          Node（npm ci + npm run build）+ Go（go build -o bin/）
  deploy start          编排启动：Node 先起 → 等 /pages 就绪 → Go 后起
  deploy stop           强杀停止（读取 .deploy/*.pid）
  deploy restart        停止后重新启动
  deploy status         进程存活 + 端口状态
  deploy logs [-n N]    tail 日志（默认 100 行）

TUI 操作：↑/↓ 选择，Enter 执行，q 退出；执行中 Ctrl+C 中断，完成后按任意键返回。
交叉编译：GOOS=linux GOARCH=amd64 go build -o deploy-linux-amd64 .
`

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run() error {
	root := core.Root()
	if root == "" {
		return fmt.Errorf("未找到仓库根（缺少 frame/go、frame/node），请在仓库内运行")
	}
	args := os.Args[1:]
	if len(args) == 0 {
		return tui.Run(root)
	}
	switch args[0] {
	case "check":
		return cmdCheck(root)
	case "config":
		return cmdConfig(root, args[1:])
	case "build":
		return core.Build(context.Background(), root, os.Stdout)
	case "start":
		return core.Start(context.Background(), root, os.Stdout)
	case "stop":
		return core.Stop(root, os.Stdout)
	case "restart":
		return core.Restart(context.Background(), root, os.Stdout)
	case "status":
		return cmdStatus(root)
	case "logs":
		return cmdLogs(root, args[1:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: "+args[0])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
		return nil
	}
}

func cmdCheck(root string) error {
	return core.Check(root, os.Stdout)
}

func cmdStatus(root string) error {
	st := core.GetStatus(root)
	line := func(name string, p core.ProcState, port int, portOK bool) {
		state := "stopped"
		if p.Alive {
			state = "running"
		}
		pid := "-"
		if p.PID > 0 {
			pid = strconv.Itoa(p.PID)
		}
		portState := "空闲"
		if portOK {
			portState = "占用"
		}
		fmt.Printf("%-6s PID %-8s %-7s :%d %s\n", name, pid, state, port, portState)
	}
	line("node", st.Node, 3000, st.Port3000)
	line("go", st.Go, 8080, st.Port8080)
	db := "不可达"
	if st.MySQL {
		db = "可达"
	}
	fmt.Printf("MySQL   3306 %s\n", db)
	if !st.Node.Alive && !st.Go.Alive {
		fmt.Println("（当前未运行）")
	}
	return nil
}

func cmdLogs(root string, args []string) error {
	n := 100
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--lines":
			if i+1 >= len(args) {
				return fmt.Errorf("%s 需要数字参数", args[i])
			}
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("%s 参数非法: %s", args[i], args[i+1])
			}
			n = v
			i++
		default:
			return fmt.Errorf("未知参数: %s", args[i])
		}
	}
	return core.PrintLogs(root, n, os.Stdout)
}

func cmdConfig(root string, args []string) error {
	envPath := filepath.Join(root, ".env.local")

	// --set 模式：追加/覆盖，保留已有键
	sets := map[string]string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--set" {
			if i+1 >= len(args) {
				return fmt.Errorf("--set 需要 KEY=VALUE 参数")
			}
			kv := args[i+1]
			eq := strings.IndexByte(kv, '=')
			if eq <= 0 {
				return fmt.Errorf("--set 参数格式应为 KEY=VALUE: %s", kv)
			}
			key := strings.TrimSpace(kv[:eq])
			if !core.ValidKey(key) {
				return fmt.Errorf("非法键名: %s", key)
			}
			sets[key] = kv[eq+1:]
			i++
			continue
		}
		return fmt.Errorf("未知参数: %s", args[i])
	}
	if len(sets) > 0 {
		f, err := core.LoadEnvFile(envPath)
		if err != nil {
			return err
		}
		for k, v := range sets {
			f.Set(k, v)
		}
		if err := f.Save(envPath); err != nil {
			return fmt.Errorf("写入 %s 失败: %w", envPath, err)
		}
		cfg, err := core.LoadConfig(root)
		if err != nil {
			return err
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		fmt.Printf("已更新 %s：\n", envPath)
		for _, l := range cfg.MaskedSummary() {
			fmt.Println("  " + l)
		}
		return nil
	}

	// 无文件 → 交互问答生成；有文件 → 展示摘要
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return configWizard(envPath)
	}
	return core.ConfigSummary(root, os.Stdout)
}

// configWizard 交互问答生成 .env.local（仅必填 DSN 强制输入，其余可回车跳过）。
func configWizard(envPath string) error {
	r := bufio.NewReader(os.Stdin)
	prompt := func(label string) string {
		fmt.Printf("%s: ", label)
		s, _ := r.ReadString('\n')
		return strings.TrimSpace(s)
	}

	f := &core.EnvFile{}
	f.AddComment("# 生成于 tools/deploy config 向导（issue #112）")
	f.AddComment("# 必填：MySQL 连接（库不存在时启动会自动建库建表）")
	fmt.Println("未找到 .env.local，开始交互式生成（回车跳过可选配置）:")
	fmt.Println("示例: root:密码@tcp(127.0.0.1:3306)/ven_blog?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci")
	var dsn string
	for dsn == "" {
		dsn = prompt("BLOG_MYSQL_DSN（必填）")
		if dsn == "" {
			fmt.Println("  BLOG_MYSQL_DSN 必填，请重试")
		}
	}
	f.Set(core.EnvDSN, dsn)

	f.AddBlank()
	f.AddComment("# 可选：种子 author 账号（首次启动用户表为空时写入）")
	if v := prompt("BLOG_AUTHOR_NAME"); v != "" {
		f.Set("BLOG_AUTHOR_NAME", v)
	}
	if v := prompt("BLOG_AUTHOR_PASSWORD"); v != "" {
		f.Set("BLOG_AUTHOR_PASSWORD", v)
	}

	f.AddBlank()
	f.AddComment("# 可选：内部令牌（Go 与 Node 两侧需一致；不配置默认 development-token）")
	if v := prompt("VEN_INTERNAL_TOKEN"); v != "" {
		f.Set(core.EnvToken, v)
	}

	f.AddBlank()
	f.AddComment("# 可选：站点 URL / AI 审核 worker（BLOG_LLM_API_KEY 未配置则不启动）")
	if v := prompt("BLOG_SITE_URL"); v != "" {
		f.Set("BLOG_SITE_URL", v)
	}
	if v := prompt("BLOG_LLM_BASE_URL"); v != "" {
		f.Set("BLOG_LLM_BASE_URL", v)
	}
	if v := prompt("BLOG_LLM_API_KEY"); v != "" {
		f.Set("BLOG_LLM_API_KEY", v)
	}
	if v := prompt("BLOG_LLM_MODEL"); v != "" {
		f.Set("BLOG_LLM_MODEL", v)
	}
	if v := prompt("BLOG_MODERATOR_INTERVAL"); v != "" {
		f.Set("BLOG_MODERATOR_INTERVAL", v)
	}
	if v := prompt("BLOG_MODERATOR_BATCH"); v != "" {
		f.Set("BLOG_MODERATOR_BATCH", v)
	}

	if err := f.Save(envPath); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", envPath, err)
	}
	fmt.Printf("已生成 %s（%d 个键）\n", envPath, len(f.Keys()))
	return nil
}
