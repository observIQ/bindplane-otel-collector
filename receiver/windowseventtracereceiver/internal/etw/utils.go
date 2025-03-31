//go:build windows
// +build windows

package etw

import (
	"crypto/rand"
	"fmt"
	"syscall"
	"unsafe"
)

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// UTF16BytesToString transforms a bytes array of UTF16 encoded characters to
// a Go string
func UTF16BytesToString(utf16 []byte) string {
	return syscall.UTF16ToString(*(*[]uint16)(unsafe.Pointer(&utf16)))
}

// UTF16PtrToString transforms a *uint16 to a Go string
func UTF16PtrToString(utf16 *uint16) string {
	return UTF16AtOffsetToString(uintptr(unsafe.Pointer(utf16)), 0)
}

func Wcslen(uintf16 *uint16) (len uint64) {
	for it := uintptr((unsafe.Pointer(uintf16))); ; it += 2 {
		wc := (*uint16)(unsafe.Pointer(it))
		if *wc == 0 {
			return
		}
		len++
	}
}

func UTF16AtOffsetToString(pstruct uintptr, offset uintptr) string {
	out := make([]uint16, 0, 64)
	wc := (*uint16)(unsafe.Pointer(pstruct + offset))
	for i := uintptr(2); *wc != 0; i += 2 {
		out = append(out, *wc)
		wc = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}
	return syscall.UTF16ToString(out)
}

func CopyData(pointer uintptr, size int) []byte {
	out := make([]byte, 0, size)
	for it := pointer; it != pointer+uintptr(size); it++ {
		b := (*byte)(unsafe.Pointer(it))
		out = append(out, *b)
	}
	return out
}

// UUID is a simple UUIDgenerator
func UUID() (uuid string, err error) {
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
