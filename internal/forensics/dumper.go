package forensics

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"syscall"
)

// DumpMemory reads a specific chunk of RAM from a live process.
func (s *Scanner) DumpMemory(pid int, region MemoryRegion) ([]byte, error) {
	// 1. Attach to the process. This pauses it and gives us permission to read its RAM.
	// Without PTRACE_ATTACH, reading /proc/PID/mem will often error or yield zeros.
	if err := syscall.PtraceAttach(pid); err != nil {
		return nil, fmt.Errorf("ptrace attach failed (Ensure CAP_SYS_PTRACE): %w", err)
	}
	defer syscall.PtraceDetach(pid)

	// Wait for the process to actually pause
	var wstatus syscall.WaitStatus
	if _, err := syscall.Wait4(pid, &wstatus, 0, nil); err != nil {
		return nil, fmt.Errorf("wait4 failed during attach: %w", err)
	}

	// 2. Open the memory file descriptor
	memPath := fmt.Sprintf("/proc/%d/mem", pid)
	memFile, err := os.Open(memPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory fd: %w", err)
	}
	defer memFile.Close()

	// 3. Seek to the start of the anomalous RWX region
	if _, err := memFile.Seek(int64(region.StartAddr), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to 0x%x failed: %w", region.StartAddr, err)
	}

	// 4. Read the chunk
	size := region.EndAddr - region.StartAddr

	// Safety limit against massive reads crashing the EDR
	if size > 1024*1024*50 { // 50 MB max per anomaly
		size = 1024 * 1024 * 50
	}

	buffer := make([]byte, size)
	n, err := io.ReadFull(memFile, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("memory read failed at 0x%x: %w", region.StartAddr, err)
	}

	return buffer[:n], nil
}

// FormatDumpToHex truncates and formats raw bytes for the LLM context limits.
func FormatDumpToHex(dump []byte, maxBytes int) string {
	if len(dump) > maxBytes {
		dump = dump[:maxBytes]
	}

	// We use standard hexdump format: "00000000  90 90 90 90 ..."
	return hex.Dump(dump)
}
