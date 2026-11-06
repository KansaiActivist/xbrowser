//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const maxDebugLogBytes = 1 << 20

var debugLogFile *os.File

func initDebugLog(baseDir string) {
	_ = os.MkdirAll(baseDir, 0o755)
	path := filepath.Join(baseDir, "debug.log")
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if st, err := os.Stat(path); err == nil && st.Size() > maxDebugLogBytes {
		flags = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err == nil {
		debugLogFile = f
	}
}

func debugLog(format string, args ...interface{}) {
	if debugLogFile == nil {
		return
	}
	fmt.Fprintf(debugLogFile, "[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
