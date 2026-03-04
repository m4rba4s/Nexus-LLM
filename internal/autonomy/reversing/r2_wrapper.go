package reversing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// R2Wrapper orchestrates headless `radare2` execution.
type R2Wrapper struct {
	binaryPath string // path to the radare2 executable
}

// NewR2Wrapper initializes the wrapper. Assumes `radare2` is in $PATH.
func NewR2Wrapper() *R2Wrapper {
	binPath, err := exec.LookPath("r2")
	if err != nil {
		fmt.Printf("[R2-WARN] radare2 (r2) not found in PATH: %v. Reversing capabilities will fail until installed.\n", err)
	}

	return &R2Wrapper{
		binaryPath: binPath,
	}
}

// FunctionDump represents a JSON-parsed disassembly block from R2 (`pdfj`).
type FunctionDump struct {
	Name string `json:"name"`
	Ops  []struct {
		Offset int64  `json:"offset"`
		Disasm string `json:"opcode"`
	} `json:"ops"`
}

// ExecuteCommand runs a radare2 command against a file and returns the raw output.
// Uses `-q` (quiet) and `-c` (command line) flags.
func (rw *R2Wrapper) ExecuteCommand(ctx context.Context, targetBinary string, r2cmd string) ([]byte, error) {
	if rw.binaryPath == "" {
		return nil, fmt.Errorf("radare2 is not installed or not in PATH")
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // Prevent analysis hangs
	defer cancel()

	// r2 -q -c "aaa; pdfj @ main" ./malware
	cmd := exec.CommandContext(execCtx, rw.binaryPath, "-q", "-c", r2cmd, targetBinary)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if execCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("r2 execution timed out analyzing %s", targetBinary)
	}
	if err != nil {
		return nil, fmt.Errorf("r2 execution failed: %w\nStderr: %s", err, errBuf.String())
	}

	return outBuf.Bytes(), nil
}

// AnalyzeMainFunction runs analysis (`aa`) and dumps `main` or `entry0` in JSON format.
func (rw *R2Wrapper) AnalyzeMainFunction(ctx context.Context, targetBinary string) (*FunctionDump, error) {
	// Execute 'aa' (basic analysis) with timeout then 'pdfj @ main' (print disassembly JSON at main)
	output, err := rw.ExecuteCommand(ctx, targetBinary, "e anal.timeout=10; aa; pdfj @ main")
	if err == nil {
		var dump FunctionDump
		if parseErr := json.Unmarshal(output, &dump); parseErr == nil && len(dump.Ops) > 0 {
			return &dump, nil
		}
	}

	// Fallback for stripped binaries or absent 'main' symbol: print disassembly at entry point
	fallbackOutput, fallbackErr := rw.ExecuteCommand(ctx, targetBinary, "e anal.timeout=10; aa; pdfj @ entry0")
	if fallbackErr != nil {
		return nil, fmt.Errorf("both 'main' and 'entry0' disassembly failed: %w", fallbackErr)
	}

	var dump FunctionDump
	if err := json.Unmarshal(fallbackOutput, &dump); err != nil {
		return nil, fmt.Errorf("failed to parse fallback r2 JSON output (%s): %w", string(fallbackOutput), err)
	}

	return &dump, nil
}
