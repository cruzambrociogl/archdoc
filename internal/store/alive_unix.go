//go:build !windows

package store

import (
	"errors"
	"syscall"
)

// alive says whether a process is running: signal 0 is delivered to no one, and fails only when
// the process is gone.
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
