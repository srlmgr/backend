//go:build unix

package tcpostgres

import (
	"os"
	"path/filepath"
	"syscall"
)

// withContainerLock serializes container creation/start across concurrent test
// binaries on this machine, working around a testcontainers-go race when multiple
// processes try to reuse/start the same named container at once.
func withContainerLock(fn func() error) error {
	lockPath := filepath.Join(os.TempDir(), "srlmgr-test-container.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	//nolint:errcheck // best-effort unlock
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return fn()
}
