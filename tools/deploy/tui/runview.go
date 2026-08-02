package tui

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// runLineMsg 执行输出的单行。
type runLineMsg struct{ line string }

// runDoneMsg 执行结束。
type runDoneMsg struct{ err error }

// RunView 执行视图：运行子命令并实时展示输出；完成后按键返回菜单，运行中可 Ctrl+C 中断。
type RunView struct {
	title       string
	lines       []string // 最多保留 2000 行
	ctx         context.Context
	cancel      context.CancelFunc
	fn          func(io.Writer) error
	done        bool
	interrupted bool
	err         error
}

// NewRunView 构造执行视图。
func NewRunView(title string, ctx context.Context, cancel context.CancelFunc, fn func(io.Writer) error) *RunView {
	return &RunView{title: title, ctx: ctx, cancel: cancel, fn: fn}
}

// start 返回启动执行的 Cmd：io.Pipe 把 fn 的输出逐行经 send 推送给程序。
func (v *RunView) start(send func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		go func() {
			pr, pw := io.Pipe()
			doneCh := make(chan error, 1)
			go func() {
				doneCh <- v.fn(pw)
				pw.Close()
			}()
			sc := bufio.NewScanner(pr)
			sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
			for sc.Scan() {
				send(runLineMsg{line: sc.Text()})
			}
			err := <-doneCh
			send(runDoneMsg{err: err})
		}()
		return nil
	}
}

// Update 处理执行视图消息：追加输出；Ctrl+C 中断；完成后任意键返回。
// 返回（是否返回菜单, 附带 Cmd）。
func (v *RunView) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case runLineMsg:
		v.lines = append(v.lines, msg.line)
		if len(v.lines) > 2000 {
			v.lines = v.lines[len(v.lines)-2000:]
		}
	case runDoneMsg:
		v.done = true
		v.err = msg.err
		if errors.Is(msg.err, context.Canceled) {
			v.interrupted = true
			v.err = nil
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if v.done {
				return true, nil
			}
			if !v.interrupted {
				v.interrupted = true
				v.cancel()
			}
		case "q", "enter", " ":
			if v.done {
				return true, nil
			}
		}
	}
	return false, nil
}

// View 渲染执行视图：标题 + 滚动输出（最后几行）+ 状态页脚。
func (v *RunView) View(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		Render("▶ " + v.title)
	limit := height - 6
	if limit < 5 {
		limit = 5
	}
	body := "  （无输出）"
	lines := v.lines
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	if len(lines) > 0 {
		body = strings.Join(lines, "\n")
	}
	var footer string
	switch {
	case v.done && v.interrupted:
		footer = "已中断（按任意键返回）"
	case v.done && v.err != nil:
		footer = "✗ 失败: " + v.err.Error() + "（按任意键返回）"
	case v.done:
		footer = "✓ 完成（按任意键返回）"
	default:
		footer = "运行中… Ctrl+C 中断"
	}
	footer = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(footer)
	return strings.Join([]string{title, "", body, "", footer}, "\n")
}
