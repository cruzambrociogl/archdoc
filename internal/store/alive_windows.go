//go:build windows

package store

import "os"

// alive says whether a process is running. Windows cannot be asked without opening the process,
// which FindProcess does: it fails when there is none.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	p.Release()
	return true
}
