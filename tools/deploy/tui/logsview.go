package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ven_hybird/tools/deploy/core"
)

// LogsView 日志视图：tail 展示，↑↓ 滚动，Tab 切换 node/go，r 刷新，q 返回。
type LogsView struct {
	root   string
	which  int // 0=node, 1=go
	lines  []string
	offset int
	errMsg string
}

// NewLogsView 构造日志视图并加载初始 tail。
func NewLogsView(root string) *LogsView {
	v := &LogsView{root: root}
	v.reload()
	return v
}

func (v *LogsView) procName() core.ProcName {
	if v.which == 1 {
		return core.ProcGo
	}
	return core.ProcNode
}

// reload 重新 tail 当前日志文件。
func (v *LogsView) reload() {
	lines, err := core.Tail(core.LogPath(v.root, v.procName()), 100)
	if err != nil {
		v.errMsg = err.Error()
		v.lines = nil
	} else {
		v.errMsg = ""
		v.lines = lines
	}
	v.offset = 0
}

// Update 处理按键，返回是否返回菜单。
func (v *LogsView) Update(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	switch key.String() {
	case "q", "ctrl+c", "esc":
		return true
	case "up", "k":
		if v.offset > 0 {
			v.offset--
		}
	case "down", "j":
		v.offset++
	case "pgup":
		v.offset -= 10
	case "pgdown":
		v.offset += 10
	case "tab":
		v.which = 1 - v.which
		v.reload()
	case "r":
		v.reload()
	}
	return false
}

// View 渲染日志视图：标题 + 滚动窗口 + 操作页脚。
func (v *LogsView) View(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		Render(fmt.Sprintf("日志 logs/%s.log（tail 100 行）", v.procName()))
	var body string
	if v.errMsg != "" {
		body = v.errMsg
	} else if len(v.lines) == 0 {
		body = "  （空日志）"
	} else {
		limit := height - 6
		if limit < 5 {
			limit = 5
		}
		maxOffset := len(v.lines) - limit
		if maxOffset < 0 {
			maxOffset = 0
		}
		if v.offset > maxOffset {
			v.offset = maxOffset
		}
		end := v.offset + limit
		if end > len(v.lines) {
			end = len(v.lines)
		}
		body = strings.Join(v.lines[v.offset:end], "\n")
	}
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Tab 切换 node/go · ↑↓ 滚动 · r 刷新 · q 返回")
	return strings.Join([]string{title, "", body, "", footer}, "\n")
}
