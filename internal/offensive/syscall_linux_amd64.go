package offensive

// Go declarations for direct syscall Assembly stubs (syscall_linux_amd64.s).
// These functions are implemented in Plan9 Assembly and linked at compile time.

// sysMemfdCreate creates an anonymous file in RAM.
// Returns the file descriptor or -1 with errno.
func sysMemfdCreate(name *byte, flags uint) (fd uintptr, errno uintptr)

// sysExecveAt executes a program referred to by a file descriptor.
// With AT_EMPTY_PATH (0x1000), pathname is ignored and fd is used directly.
// Only returns on failure.
func sysExecveAt(fd uintptr, pathname *byte, argv **byte, envp **byte, flags uintptr) (errno uintptr)

// sysWrite writes data to a file descriptor via direct syscall.
func sysWrite(fd uintptr, buf *byte, count uintptr) (n uintptr, errno uintptr)
