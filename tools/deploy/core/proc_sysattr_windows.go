//go:build windows

package core

import "syscall"

// Windows 进程创建标志（winbase.h）。syscall 包未导出，这里按 SDK 数值定义：
// CREATE_NEW_PROCESS_GROUP (0x200)——子进程自成一族，Ctrl+C 等控制台事件不波及；
// DETACHED_PROCESS (0x8)——脱离父终端控制台，关闭终端窗口不杀进程。
// HideWindow 由 Go 内部转成 CREATE_NO_WINDOW，避免启动时闪出控制台窗口。
const (
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
)

// sysProcAttr 返回 Windows 子进程 detach 属性（详见文件头注释）。
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | detachedProcess,
		HideWindow:    true,
	}
}
