//go:build windows

package discovery

import (
	"os"
	"syscall"
	"time"
)

// GetFileTimes extracts both ModifiedAt and CreatedAt timestamps from the file.
func GetFileTimes(path string, info os.FileInfo) (time.Time, time.Time) {
	modTime := info.ModTime()
	createdTime := modTime

	if sys, ok := info.Sys().(*syscall.Win32FileAttributeData); ok && sys != nil {
		createdTime = time.Unix(0, sys.CreationTime.Nanoseconds())
	}

	return modTime, createdTime
}
