// Package tui 提供 bubbletea 主界面：状态面板 + 菜单 + 执行视图 + 日志视图。
package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ven_hybird/tools/deploy/core"
)

// Action 菜单动作。
type Action int

const (
	ActionNone Action = iota
	ActionCheck
	ActionConfig
	ActionBuild
	ActionStart
	ActionStop
	ActionRestart
	ActionLogs
	ActionQuit
)

// MenuItem 菜单项。
type MenuItem struct {
	Action Action
	Label  string
	Hint   string
}

// MenuItems 主菜单（TUI 与单测共用）。
var MenuItems = []MenuItem{
	{ActionCheck, "检测", "环境检测：go/node/npm/MySQL/端口"},
	{ActionConfig, "配置", "查看 .env.local（生成用 config 子命令）"},
	{ActionBuild, "构建", "Node npm ci+build → Go build"},
	{ActionStart, "启动", "Node 先起、等就绪、Go 后起"},
	{ActionStop, "停止", "强杀进程（读 .deploy/*.pid）"},
	{ActionRestart, "重启", "停止后重新启动"},
	{ActionLogs, "日志", "tail 查看 node/go 日志"},
	{ActionQuit, "退出", "q 或 Ctrl+C"},
}

// Menu 是主菜单的纯状态（不依赖 TUI 运行时，可单测）。
type Menu struct {
	Items    []MenuItem
	Cursor   int
	Selected Action // 最近一次 Enter 选中的动作；ActionNone 表示未选中
}

// NewMenu 返回初始菜单。
func NewMenu() Menu { return Menu{Items: MenuItems} }

// Update 纯函数：处理按键，返回（新菜单, 触发动作, 是否退出）。
// 方向键/jk 移动光标；Enter/空格选中（置 Selected 标记）；q/Ctrl+C 退出。
func (m Menu) Update(msg tea.Msg) (Menu, Action, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, ActionNone, false
	}
	switch key.String() {
	case "up", "k":
		if len(m.Items) > 0 {
			m.Cursor = (m.Cursor - 1 + len(m.Items)) % len(m.Items)
		}
	case "down", "j":
		if len(m.Items) > 0 {
			m.Cursor = (m.Cursor + 1) % len(m.Items)
		}
	case "enter", " ":
		act := m.Items[m.Cursor].Action
		m.Selected = act
		return m, act, act == ActionQuit
	case "q", "ctrl+c":
		return m, ActionQuit, true
	}
	return m, ActionNone, false
}

// PanelStatus 顶部状态面板的数据。
type PanelStatus struct {
	NodeBuilt bool // frame/node/dist/main.js 存在
	GoBuilt   bool // bin/ven_hybird[.exe] 存在
	EnvOK     bool // .env.local 存在且 DSN 已配置
	Node      core.ProcState
	Go        core.ProcState
	Port3000  bool
	Port8080  bool
	MySQL     bool
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// collectPanelStatus 采集状态面板数据（文件/端口探测，无子进程调用）。
func collectPanelStatus(root string) PanelStatus {
	s := core.GetStatus(root)
	p := PanelStatus{
		NodeBuilt: fileExists(filepath.Join(root, "frame", "node", "dist", "main.js")),
		GoBuilt:   fileExists(core.BinPath(root)),
		Node:      s.Node,
		Go:        s.Go,
		Port3000:  s.Port3000,
		Port8080:  s.Port8080,
		MySQL:     s.MySQL,
	}
	if c, err := core.LoadConfig(root); err == nil {
		p.EnvOK = c.Get(core.EnvDSN) != ""
	}
	return p
}

// phase 界面阶段。
type phase int

const (
	phaseMenu phase = iota
	phaseRun
	phaseLogs
)

// statusTickMsg 定时刷新状态面板。
type statusTickMsg struct{}

// Model 是 TUI 主模型。
type Model struct {
	root    string
	program *tea.Program
	phase   phase
	menu    Menu
	panel   PanelStatus
	run     *RunView
	logs    *LogsView
	width   int
	height  int
}

// New 构造主模型。
func New(root string) *Model {
	return &Model{root: root, menu: NewMenu(), panel: collectPanelStatus(root)}
}

// Run 启动 TUI（阻塞直到退出）。
func Run(root string) error {
	m := New(root)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd { return m.tickCmd() }

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return statusTickMsg{} })
}

// Update 分派消息：菜单阶段按键 → 纯函数 menu.Update；执行/日志阶段 → 子视图。
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case statusTickMsg:
		if m.phase == phaseMenu {
			m.panel = collectPanelStatus(m.root)
		}
		return m, m.tickCmd()
	case tea.KeyMsg:
		switch m.phase {
		case phaseRun:
			if m.run != nil {
				back, cmd := m.run.Update(msg)
				if back {
					m.backToMenu()
				}
				return m, cmd
			}
		case phaseLogs:
			if m.logs != nil && m.logs.Update(msg) {
				m.backToMenu()
			}
			return m, nil
		default: // phaseMenu
			menu, act, quit := m.menu.Update(msg)
			m.menu = menu
			if quit {
				return m, tea.Quit
			}
			if act != ActionNone {
				return m, m.beginAction(act)
			}
			return m, nil
		}
	}
	// runLineMsg / runDoneMsg 等运行时消息
	if m.phase == phaseRun && m.run != nil {
		back, cmd := m.run.Update(msg)
		if back {
			m.backToMenu()
		}
		return m, cmd
	}
	return m, nil
}

// beginAction 进入执行/日志视图。
func (m *Model) beginAction(act Action) tea.Cmd {
	if act == ActionLogs {
		m.logs = NewLogsView(m.root)
		m.phase = phaseLogs
		return nil
	}
	title, fn := m.actionRunner(act)
	if fn == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.run = NewRunView(title, ctx, cancel, fn)
	m.phase = phaseRun
	return m.run.start(m.program.Send)
}

// backToMenu 回到菜单并刷新状态面板。
func (m *Model) backToMenu() {
	m.phase = phaseMenu
	m.panel = collectPanelStatus(m.root)
}

// actionRunner 动作 → 执行函数（注入 ctx 供 Ctrl+C 中断）。
func (m *Model) actionRunner(act Action) (string, func(io.Writer) error) {
	for _, it := range MenuItems {
		if it.Action != act {
			continue
		}
		switch act {
		case ActionCheck:
			return it.Label, func(w io.Writer) error { return core.Check(m.root, w) }
		case ActionConfig:
			return it.Label, func(w io.Writer) error { return core.ConfigSummary(m.root, w) }
		case ActionBuild:
			return it.Label, func(w io.Writer) error { return core.Build(m.ctxOf(), m.root, w) }
		case ActionStart:
			return it.Label, func(w io.Writer) error { return core.Start(m.ctxOf(), m.root, w) }
		case ActionStop:
			return it.Label, func(w io.Writer) error { return core.Stop(m.root, w) }
		case ActionRestart:
			return it.Label, func(w io.Writer) error { return core.Restart(m.ctxOf(), m.root, w) }
		}
	}
	return "", nil
}

// ctxOf 返回当前 RunView 的 ctx（动作执行期间可用）。
func (m *Model) ctxOf() context.Context {
	if m.run != nil {
		return m.run.ctx
	}
	return context.Background()
}

// View 渲染当前阶段。
func (m *Model) View() string {
	switch m.phase {
	case phaseRun:
		if m.run != nil {
			return m.run.View(m.width, m.height)
		}
	case phaseLogs:
		if m.logs != nil {
			return m.logs.View(m.width, m.height)
		}
	}
	return m.menuView()
}

func (m *Model) menuView() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		Render("ven-blog 部署工具（tools/deploy）")
	panel := m.panelView()
	menu := m.menuBody()
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("↑/↓ 选择 · Enter 执行 · q/Ctrl+C 退出")
	return strings.Join([]string{title, "", panel, "", menu, "", footer}, "\n")
}

func (m *Model) panelView() string {
	p := m.panel
	mark := func(b bool) string {
		if b {
			return "✓"
		}
		return "✗"
	}
	state := func(alive bool, pid int) string {
		if alive {
			return fmt.Sprintf("running (PID %d)", pid)
		}
		return "stopped"
	}
	port := func(b bool) string {
		if b {
			return "占用"
		}
		return "空闲"
	}
	db := "✗ 不可达"
	if p.MySQL {
		db = "✓ 可达"
	}
	env := "✗ 缺失/未配 DSN"
	if p.EnvOK {
		env = "✓ 已配置"
	}
	lines := []string{
		fmt.Sprintf("构建产物   Node dist/main.js %s    Go bin/%s %s", mark(p.NodeBuilt), core.BinName(), mark(p.GoBuilt)),
		fmt.Sprintf("运行状态   Node %s    Go %s", state(p.Node.Alive, p.Node.PID), state(p.Go.Alive, p.Go.PID)),
		fmt.Sprintf("端口       :3000 %s    :8080 %s    MySQL 3306 %s", port(p.Port3000), port(p.Port8080), db),
		fmt.Sprintf("配置       .env.local %s", env),
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (m *Model) menuBody() string {
	var b strings.Builder
	for i, item := range m.menu.Items {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.menu.Cursor {
			cursor = "> "
			style = style.Bold(true).Foreground(lipgloss.Color("212"))
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%-6s %s", cursor, item.Label, item.Hint)))
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}
