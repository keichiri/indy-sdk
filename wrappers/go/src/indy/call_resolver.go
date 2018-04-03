package indy

// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

var (
	ErrTimeout                     = errors.New("Call to libindy timed out")
	ErrSymbol                      = errors.New("Failed to resolve symbol")
	ErrInvalidHandle               = errors.New("No such handle")
	resolver         *callResolver = nil
)

func init() {
	var c_lib_name *C.char = C.CString("libindy.so")
	handle := C.dlopen(c_lib_name, C.RTLD_LAZY)
	if handle == nil {
		panic("Failed to load libindy.so")
	}

	resolver = newCallResolver(handle)
}

type callResolver struct {
	mx             sync.Mutex
	libindyHandle  unsafe.Pointer
	indyFunctions  map[string]unsafe.Pointer
	counter        int32
	resultChannels map[int32]chan interface{}
}

func newCallResolver(handle unsafe.Pointer) *callResolver {
	return &callResolver{
		libindyHandle:  handle,
		indyFunctions:  make(map[string]unsafe.Pointer),
		resultChannels: make(map[int32]chan interface{}),
	}
}

func (r *callResolver) RegisterCall(name string) (unsafe.Pointer, int32, chan interface{}, error) {
	defer r.mx.Unlock()
	r.mx.Lock()

	pointer, exists := r.indyFunctions[name]
	if !exists {
		pointer = C.dlsym(r.libindyHandle, C.CString(name))
		if pointer == nil {
			return nil, 0, nil, ErrSymbol
		}
		r.indyFunctions[name] = pointer
	}

	resCh := make(chan interface{}, 1)
	commandHandle := r.counter
	r.counter++
	r.resultChannels[commandHandle] = resCh

	return pointer, commandHandle, resCh, nil
}

func (r *callResolver) DeregisterCall(commandHandle int32) (chan interface{}, error) {
	defer r.mx.Unlock()
	r.mx.Lock()
	resCh, exists := r.resultChannels[commandHandle]
	if !exists {
		return nil, ErrInvalidHandle
	}

	return resCh, nil
}
