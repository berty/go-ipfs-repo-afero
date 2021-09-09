//go:build !plan9 && !windows
// +build !plan9,!windows

package lock

import (
	"strings"
	"syscall"
)

func lockedByOthers(err error) bool {
	return err == syscall.EAGAIN ||
		strings.Contains(err.Error(), "resource temporarily unavailable") ||
		(strings.Contains(err.Error(), "already locked") && !strings.Contains(err.Error(), "by us"))
}
