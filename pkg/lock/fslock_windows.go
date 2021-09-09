package lock

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows"
)

func lockedByOthers(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		strings.Contains(err.Error(), "being used by another process") ||
		(strings.Contains(err.Error(), "already locked") && !strings.Contains(err.Error(), "by us"))
}
