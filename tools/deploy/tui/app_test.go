package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestMenuNavigation(t *testing.T) {
	m := NewMenu()
	if m.Cursor != 0 {
		t.Fatalf("初始光标 = %d, want 0", m.Cursor)
	}

	// ↓ 两次
	m, act, quit := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 1 || act != ActionNone || quit {
		t.Errorf("↓ 后 cursor=%d act=%v quit=%v", m.Cursor, act, quit)
	}
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 2 {
		t.Errorf("第二次 ↓ 后 cursor = %d, want 2", m.Cursor)
	}

	// ↑ 一次
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 1 {
		t.Errorf("↑ 后 cursor = %d, want 1", m.Cursor)
	}

	// 循环回绕：连按 ↑ 到末尾
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 0 {
		t.Errorf("回绕 0: cursor = %d", m.Cursor)
	}
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if want := len(m.Items) - 1; m.Cursor != want {
		t.Errorf("回绕末尾: cursor = %d, want %d", m.Cursor, want)
	}

	// j/k 快捷键
	m, _, _ = m.Update(keyRunes('k'))
	if want := len(m.Items) - 2; m.Cursor != want {
		t.Errorf("k 后 cursor = %d, want %d", m.Cursor, want)
	}
	m, _, _ = m.Update(keyRunes('j'))
	if want := len(m.Items) - 1; m.Cursor != want {
		t.Errorf("j 后 cursor = %d, want %d", m.Cursor, want)
	}
}

func TestMenuSelectSetsMarker(t *testing.T) {
	m := NewMenu()
	// 移动到"构建"（index 2）
	for i := 0; i < 2; i++ {
		m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.Items[m.Cursor].Action != ActionBuild {
		t.Fatalf("光标应停在构建项, 实际 %+v", m.Items[m.Cursor])
	}
	m, act, quit := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if act != ActionBuild {
		t.Errorf("Enter 应触发 ActionBuild, got %v", act)
	}
	if m.Selected != ActionBuild {
		t.Errorf("Selected 标记 = %v, want ActionBuild", m.Selected)
	}
	if quit {
		t.Error("选中非退出项不应退出")
	}

	// 空格同样选中
	m2, act2, _ := NewMenu().Update(tea.KeyMsg{Type: tea.KeySpace})
	if act2 != ActionCheck || m2.Selected != ActionCheck {
		t.Errorf("空格选中 = %v/%v, want ActionCheck", act2, m2.Selected)
	}
}

func TestMenuQuitKeys(t *testing.T) {
	// q
	_, act, quit := NewMenu().Update(keyRunes('q'))
	if !quit || act != ActionQuit {
		t.Errorf("q: quit=%v act=%v", quit, act)
	}
	// Ctrl+C
	_, act, quit = NewMenu().Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !quit || act != ActionQuit {
		t.Errorf("Ctrl+C: quit=%v act=%v", quit, act)
	}
	// 在"退出"项按 Enter
	m := NewMenu()
	m.Cursor = len(m.Items) - 1
	_, act, quit = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !quit || act != ActionQuit {
		t.Errorf("退出项 Enter: quit=%v act=%v", quit, act)
	}
}

func TestMenuIgnoresOtherMsgs(t *testing.T) {
	m, act, quit := NewMenu().Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if act != ActionNone || quit || m.Cursor != 0 {
		t.Errorf("非按键消息不应影响菜单: act=%v quit=%v cursor=%d", act, quit, m.Cursor)
	}
}
