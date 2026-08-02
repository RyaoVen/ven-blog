//go:build unix

package core

import "syscall"

// sysProcAttr 返回 POSIX 子进程 detach 属性：
// Setpgid 让子进程自成一进程组（信号/终端事件不波及）。
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
