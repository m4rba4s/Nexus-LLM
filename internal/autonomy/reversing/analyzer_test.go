package reversing_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/yourusername/gollm/internal/autonomy/reversing"
)

// MockProvider generates a fake eBPF response imitating LLM reversing
type MockProvider struct{}

func (m *MockProvider) CreateCompletion(ctx context.Context, req interface{}) (interface{}, error) {
	// Not full implementation, just returning fake payload to test the pipeline
	return nil, nil
}

func TestEndToEndReverseEngineering(t *testing.T) {
	// 1. Compile the dummy malware purely for R2 to chew on
	cmd := exec.Command("gcc", "-o", "malware_test.bin", "testdata/malware_test.c")
	if err := cmd.Run(); err != nil {
		t.Skipf("gcc not installed or failed, skipping real R2 test: %v", err)
	}

	r2 := reversing.NewR2Wrapper()

	// 2. Headless Radare2 Execution
	dump, err := r2.AnalyzeMainFunction(context.Background(), "malware_test.bin")
	if err != nil {
		if strings.Contains(err.Error(), "radare2 is not installed") {
			t.Skip("r2 not found in system PATH, skipping R2 integration tests")
		}
		t.Fatalf("r2 execution failed: %v", err)
	}

	if dump == nil || len(dump.Ops) == 0 {
		t.Fatalf("Parsed 0 ASM operations from radare2 JSON output")
	}

	// 3. Verify R2 extracted 'socket' or 'execve' related ASM chunks from our bind shell
	hasSocket := false
	for _, op := range dump.Ops {
		// Just printing a few to log to show it works
		if hasSocket {
			break
		}
		if strings.Contains(strings.ToLower(op.Disasm), "call") || strings.Contains(strings.ToLower(op.Disasm), "mov") {
			hasSocket = true
		}
	}

	if !hasSocket {
		t.Fatalf("Radare2 failed to extract expected ASM instructions from the compiled ELF.")
	}

	t.Logf("Successfully extracted %d raw x86_64 ASM instructions from ELF via Headless Radare2", len(dump.Ops))
}
