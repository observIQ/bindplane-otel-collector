//go:build darwin

package logging

/*
#include "oslog.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

type osLogSyncer struct{}

func (s *osLogSyncer) Write(p []byte) (n int, err error) {
	cStr := C.CString(string(p))
	defer C.free(unsafe.Pointer(cStr))
	C.logMessage(cStr)
	return len(p), nil
}
