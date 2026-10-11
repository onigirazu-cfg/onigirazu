//go:build !windows

package expression

import (
	"os"
	"syscall"
)

// deviceOf is the device a file lives on, for the mount test
func deviceOf(info os.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return deviceNumber(st.Dev)
	}
	return 0
}

// deviceNumber widens Stat_t.Dev, whose type differs by platform (int32 on
// darwin, uint64 on linux), without a conversion lint has to judge
func deviceNumber(dev interface{}) uint64 {
	switch d := dev.(type) {
	case uint64:
		return d
	case int64:
		return uint64(d) // #nosec G115 -- a device number
	case uint32:
		return uint64(d)
	case int32:
		return uint64(d) // #nosec G115 -- a device number
	case uint16:
		return uint64(d)
	}
	return 0
}
