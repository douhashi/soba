//go:build windows
// +build windows

package service

import "syscall"

const CREATE_NEW_PROCESS_GROUP = 0x00000200

// getSysProcAttr returns Windows-specific process attributes for daemon
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NEW_PROCESS_GROUP,
		// This creates a new process group on Windows
	}
}
