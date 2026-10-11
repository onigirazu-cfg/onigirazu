//go:build !windows

package expression

import (
	"os"
	"syscall"
)

// deviceOf is the device a file lives on, for the mount test
func deviceOf(info os.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev) // #nosec G115 -- a device number
	}
	return 0
}
