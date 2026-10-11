//go:build windows

package expression

import "os"

func deviceOf(os.FileInfo) uint64 { return 0 }
