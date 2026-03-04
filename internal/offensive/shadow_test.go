package offensive

import (
	"fmt"
	"runtime"
	"syscall"
	"testing"
)

func TestShadowExecValidation(t *testing.T) {
	t.Run("RejectsEmptyPayload", func(t *testing.T) {
		err := ShadowExec([]byte{}, []string{"test"}, nil)
		if err == nil {
			t.Fatal("expected error for empty payload")
		}
	})

	t.Run("RejectsInvalidMagic", func(t *testing.T) {
		err := ShadowExec([]byte{0x00, 0x00, 0x00, 0x00}, []string{"test"}, nil)
		if err == nil {
			t.Fatal("expected error for invalid ELF magic")
		}
	})

	t.Run("RejectsShortPayload", func(t *testing.T) {
		err := ShadowExec([]byte{0x7f}, []string{"test"}, nil)
		if err == nil {
			t.Fatal("expected error for short payload")
		}
	})
}

func TestShadowLoadValidation(t *testing.T) {
	t.Run("RejectsInvalidPayload", func(t *testing.T) {
		_, _, err := ShadowLoad([]byte{0x00})
		if err == nil {
			t.Fatal("expected error for invalid payload")
		}
	})
}

func TestMemfdCreateDirect(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("direct syscalls require linux/amd64")
	}

	emptyName := [1]byte{0}
	fd, errno := sysMemfdCreate(&emptyName[0], mfdCloexec)
	if errno != 0 {
		t.Fatalf("sysMemfdCreate failed with errno %d", errno)
	}
	if fd == 0 {
		t.Fatal("sysMemfdCreate returned fd=0, expected >0")
	}

	data := []byte("SHADOW_EXEC_TEST_DATA")
	n, wErrno := sysWrite(fd, &data[0], uintptr(len(data)))
	if wErrno != 0 {
		t.Fatalf("sysWrite failed with errno %d", wErrno)
	}
	if n != uintptr(len(data)) {
		t.Fatalf("sysWrite: expected %d bytes, wrote %d", len(data), n)
	}

	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	t.Logf("[PASS] memfd_create → fd=%d, path=%s, wrote %d bytes", fd, path, n)

	syscall.Close(int(fd))
}

func TestIoUringEvasionPoC(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("io_uring requires Linux")
	}

	result, err := IoUringEvasionPoC("127.0.0.1:9999")
	if err != nil {
		t.Fatalf("IoUringEvasionPoC returned error: %v", err)
	}

	if result.Connected {
		t.Logf("[EVASION CONFIRMED] %s", result.ErrorMessage)
	} else {
		t.Logf("[EVASION PARTIAL] io_uring_setup unavailable: %s", result.ErrorMessage)
	}
}

func TestShadowStackSwap(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("shadow stack requires linux/amd64")
	}

	executed := false
	config := DefaultShadowConfig()

	err := WithShadowStack(config, func() {
		executed = true
	})
	if err != nil {
		t.Fatalf("WithShadowStack failed: %v", err)
	}
	if !executed {
		t.Fatal("target function was not executed on shadow stack")
	}
	t.Log("[PASS] Function executed on fabricated shadow call stack")
}

func TestMmapAnon(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("mmap requires Linux")
	}

	addr, err := mmapAnon(4096)
	if err != nil {
		t.Fatalf("mmapAnon failed: %v", err)
	}
	if addr == 0 {
		t.Fatal("mmapAnon returned NULL")
	}
	munmapAnon(addr, 4096)
	t.Logf("[PASS] mmap/munmap cycle at addr=0x%x", addr)
}
