#include "textflag.h"

// Direct syscall stubs for Shadow Execution (Phase 26).
// Bypasses Go runtime's syscall wrappers and any libc inline hooks.
// All syscalls go through the SYSCALL instruction directly.

// func sysMemfdCreate(name *byte, flags uint) (fd uintptr, errno uintptr)
// SYS_memfd_create = 319
TEXT ·sysMemfdCreate(SB), NOSPLIT, $0-32
	MOVQ name+0(FP), DI     // arg1: const char *name
	MOVQ flags+8(FP), SI    // arg2: unsigned int flags
	MOVQ $319, AX            // syscall number: memfd_create
	SYSCALL
	CMPQ AX, $-4095
	JLS  ok_memfd
	NEGQ AX
	MOVQ $-1, fd+16(FP)
	MOVQ AX, errno+24(FP)
	RET
ok_memfd:
	MOVQ AX, fd+16(FP)
	MOVQ $0, errno+24(FP)
	RET

// func sysExecveAt(fd uintptr, pathname *byte, argv **byte, envp **byte, flags uintptr) (errno uintptr)
// SYS_execveat = 322
TEXT ·sysExecveAt(SB), NOSPLIT, $0-48
	MOVQ fd+0(FP), DI           // arg1: int dirfd
	MOVQ pathname+8(FP), SI     // arg2: const char *pathname (empty string for fd-based exec)
	MOVQ argv+16(FP), DX        // arg3: char *const argv[]
	MOVQ envp+24(FP), R10       // arg4: char *const envp[]
	MOVQ flags+32(FP), R8       // arg5: int flags (AT_EMPTY_PATH = 0x1000)
	MOVQ $322, AX                // syscall number: execveat
	SYSCALL
	NEGQ AX                     // execveat only returns on error
	MOVQ AX, errno+40(FP)
	RET

// func sysWrite(fd uintptr, buf *byte, count uintptr) (n uintptr, errno uintptr)
// SYS_write = 1
TEXT ·sysWrite(SB), NOSPLIT, $0-40
	MOVQ fd+0(FP), DI
	MOVQ buf+8(FP), SI
	MOVQ count+16(FP), DX
	MOVQ $1, AX
	SYSCALL
	CMPQ AX, $-4095
	JLS  ok_write
	NEGQ AX
	MOVQ $-1, n+24(FP)
	MOVQ AX, errno+32(FP)
	RET
ok_write:
	MOVQ AX, n+24(FP)
	MOVQ $0, errno+32(FP)
	RET
