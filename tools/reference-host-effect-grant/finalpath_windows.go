//go:build windows

package main

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var procGetFinalPathNameByHandleW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

// finalPath returns the real location of an open file. Unlike filepath.
// EvalSymlinks it follows directory junctions and every other reparse point,
// because the OS reports where the handle actually lives.
func finalPath(f *os.File) (string, error) {
	buf := make([]uint16, 32768)
	n, _, _ := procGetFinalPathNameByHandleW.Call(f.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 || int(n) >= len(buf) {
		return "", errors.New("final path unavailable")
	}
	p := syscall.UTF16ToString(buf[:n])
	switch {
	case strings.HasPrefix(p, `\?\UNC\`):
		p = `\` + p[len(`\?\UNC\`):]
	case strings.HasPrefix(p, `\?\`):
		p = p[len(`\?\`):]
	}
	return p, nil
}
