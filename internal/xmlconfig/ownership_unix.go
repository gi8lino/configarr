//go:build unix

package xmlconfig

import (
	"fmt"
	"os"
	"syscall"
)

// preserveOwnership keeps the source file's owner and group before atomic replacement.
func preserveOwnership(file *os.File, original os.FileInfo) error {
	owner, ok := original.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("source ownership is unavailable")
	}

	current, err := file.Stat()
	if err != nil {
		return err
	}
	actual, ok := current.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("temporary file ownership is unavailable")
	}

	if sameOwnership(actual, owner) {
		return nil
	}
	return file.Chown(int(owner.Uid), int(owner.Gid))
}

// sameOwnership compares the user and group IDs of two Unix files.
func sameOwnership(left, right *syscall.Stat_t) bool {
	return left.Uid == right.Uid && left.Gid == right.Gid
}
