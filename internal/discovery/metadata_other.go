//go:build !windows

package discovery

import (
	"os"
	"time"
)

// GetFileTimes returns ModifiedAt and a fallback CreatedAt for non-Windows platforms.
func GetFileTimes(path string, info os.FileInfo) (time.Time, time.Time) {
	modTime := info.ModTime()
	return modTime, modTime
}
