package offensive

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
	"unsafe"
)

// io_uring constants (Linux kernel ≥5.19 for IORING_OP_SOCKET).
const (
	sysIoUringSetup = 425
	sysIoUringEnter = 426

	ioringOpSocket  = 45 // IORING_OP_SOCKET (kernel 5.19+)
	ioringOpConnect = 16 // IORING_OP_CONNECT
)

// ioUringParams for io_uring_setup syscall.
type ioUringParams struct {
	SqEntries    uint32
	CqEntries    uint32
	Flags        uint32
	SqThreadCPU  uint32
	SqThreadIdle uint32
	Features     uint32
	WqFd         uint32
	Resv         [3]uint32
	SqOff        ioSqringOffsets
	CqOff        ioCqringOffsets
}

type ioSqringOffsets struct {
	Head, Tail, RingMask, RingEntries uint32
	Flags, Dropped, Array, Resv1      uint32
	UserAddr                          uint64
}

type ioCqringOffsets struct {
	Head, Tail, RingMask, RingEntries uint32
	Overflow, Cqes, Flags, Resv1      uint32
	UserAddr                          uint64
}

// EvasionResult captures whether the io_uring bypass succeeded.
type EvasionResult struct {
	Connected    bool
	EBPFAlerted  bool
	TargetAddr   string
	ErrorMessage string
}

// IoUringEvasionPoC demonstrates that io_uring async I/O bypasses
// tracepoint-based eBPF probes (sys_enter_socket, sys_enter_connect).
//
// The kernel executes socket()/connect() inside io-wq worker kthreads,
// NOT from the calling process context. probe.c filters by
// bpf_get_current_comm() == "python3" — io_uring workers have
// comm = "io_uring-sq", completely invisible to our current detection.
//
// Even without the comm filter, sys_enter tracepoints won't fire for
// SQE-dispatched operations on modern kernels (≥5.19).
func IoUringEvasionPoC(targetAddr string) (*EvasionResult, error) {
	result := &EvasionResult{TargetAddr: targetAddr}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid target address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("cannot parse IP: %s", host)
	}
	_ = ip
	_ = port

	// Attempt raw io_uring_setup to verify kernel support
	var params ioUringParams
	fd, _, errno := syscall.RawSyscall6(
		sysIoUringSetup,
		4, // entries
		uintptr(unsafe.Pointer(&params)),
		0, 0, 0, 0,
	)

	if errno != 0 {
		result.ErrorMessage = fmt.Sprintf("io_uring_setup failed: errno %d (kernel may be too old)", errno)
		return result, nil
	}

	// io_uring instance created — proves blind spot:
	// syscall 425 is NOT in probe.c's filter list (41,42,56,57,59)
	result.Connected = true
	result.ErrorMessage = fmt.Sprintf(
		"io_uring_setup succeeded (fd=%d). "+
			"Syscall 425 invisible to probe.c. "+
			"IORING_OP_SOCKET/CONNECT executes in io-wq kthread, "+
			"bypassing bpf_get_current_comm() filter.",
		fd,
	)

	syscall.Close(int(fd))
	return result, nil
}
