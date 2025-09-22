//go:build !windows
// +build !windows

package service

import "syscall"

// getSysProcAttr returns Unix-specific process attributes for daemon
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // Create new session (become session leader)
		// This detaches the process from the terminal
	}
}
