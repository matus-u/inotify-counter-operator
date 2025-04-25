package watcher

/*
#cgo CFLAGS: -Wall
#cgo LDFLAGS:
#include <stdlib.h>
#include "filecounter.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

var watcherLock sync.Mutex
var running bool

func StartWatcher(path string) {
	watcherLock.Lock()
	defer watcherLock.Unlock()

	if running {
		return
	}
	go func() {
		cPath := C.CString(path)
		C.start_watching(cPath)
		C.free(unsafe.Pointer(cPath))
	}()
	running = true
}

func StopWatcher() {
	watcherLock.Lock()
	defer watcherLock.Unlock()

	if running {
		C.stop_watching()
		running = false
	}
}
