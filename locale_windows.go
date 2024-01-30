//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	pGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

func GetUserDefaultLocaleName(buffer *uint16, bufferLen uint32) (uint32, error) {
	r0, _, e0 := syscall.SyscallN(pGetUserDefaultLocaleName.Addr(), uintptr(unsafe.Pointer(buffer)), uintptr(bufferLen))
	if r0 == 0 {
		return 0, errnoErr(e0)
	}
	return uint32(r0), nil
}

func defaultLocaleName() string {
	n := uint32(32)
	var err error
	for {
		buf := make([]uint16, n)
		n, err = GetUserDefaultLocaleName(&buf[0], uint32(len(buf)))
		if err != nil {
			return ""
		}
		if n <= uint32(len(buf)) {
			return windows.UTF16ToString(buf[:n])
		}
	}
}
