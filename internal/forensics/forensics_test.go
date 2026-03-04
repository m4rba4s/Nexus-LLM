package forensics_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/gollm/internal/forensics"
)

// MockYARASynthesizer generates a fake JSON response imitating Claude Opus
type MockYARAProvider struct{}

func (m *MockYARAProvider) CreateCompletion(ctx context.Context, req interface{}) (interface{}, error) {
	// Not used in this test since we just want to verify the RWX extraction
	return nil, nil
}

func TestMemoryRWXAnomalyDetection(t *testing.T) {
	// 0. Compile the anomaly test binary
	compileCmd := exec.Command("gcc", "-o", "anomaly_test.bin", "testdata/anomaly_test.c")
	if err := compileCmd.Run(); err != nil {
		t.Fatalf("Failed to compile anomaly test script: %v", err)
	}

	// 1. Start the rogue process that allocates RWX memory
	cmd := exec.Command("./anomaly_test.bin")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start anomaly_test. Did you compile it? %v", err)
	}
	defer cmd.Process.Kill()

	// Let it map memory
	time.Sleep(500 * time.Millisecond)

	// 2. Init Scanner
	scanner := forensics.NewScanner()

	// 3. Scan the maps of the injected process
	anomalies, err := scanner.FindAnomalousRegions(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("Scanner failed reading /proc/maps: %v", err)
	}

	if len(anomalies) == 0 {
		t.Fatalf("Scanner failed to detect the injected Anonymous RWX region!")
	}

	t.Logf("Successfully detected %d RWX Anomaly Region(s)", len(anomalies))

	// 4. Test Ptrace Dumper
	targetRegion := anomalies[0]
	dump, err := scanner.DumpMemory(cmd.Process.Pid, targetRegion)
	if err != nil {
		t.Fatalf("Ptrace memory dump failed: %v", err)
	}

	if len(dump) == 0 {
		t.Fatalf("Memory dump is empty!")
	}

	// 5. Verify the dumped bytes match our injected NOP sled shellcode
	hexDump := forensics.FormatDumpToHex(dump, 128)
	t.Logf("Extracted Hex Dump Snippet:\n%s", hexDump)

	// In anomaly_test.c we injected \x90\x90\x90\x90\xcc\xcc\xcc\xcc
	if !strings.Contains(hexDump, "90 90 90 90 cc cc cc cc") {
		t.Fatalf("Dumped memory did not contain the injected shellcode pattern!")
	}

	t.Logf("Ptrace successfully extracted the raw injected shellcode bytes from live RAM!")
}
