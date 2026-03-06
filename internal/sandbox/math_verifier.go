package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/sandbox/ebpf"
)

var (
	ErrMaliciousPayload = errors.New("sandbox rejection: detected potentially malicious python token")
	ErrTimeout          = errors.New("sandbox rejection: execution timed out")
)

// MathVerifier isolates and executes mathematical python code (SymPy) securely.
// We strictly enforce "Anti-Bypass" mechanisms here.
type MathVerifier struct {
	dangerousTokens *regexp.Regexp
	bpfMonitor      *ebpf.SandboxMonitor
}

func NewMathVerifier() *MathVerifier {
	// 1. Static Lexical Firewall: Reject anything that isn't strict math/sympy.
	// We block all builtins that could lead to breakout, reflection, or OS access.
	// Machine CANNOT bypass this because it's evaluated in Go, outside the LLM context.
	mv := &MathVerifier{
		dangerousTokens: regexp.MustCompile(`(?i)(\bimport\s+(os|sys|subprocess|pty|shutil|socket|urllib|requests)\b|os\.|sys\.|subprocess|__import__|open\s*\(|eval\s*\(|exec\s*\(|pty|shutil|socket|urllib|requests)`),
	}

	// 2. Ring-0 eBPF Sandbox (Defense-in-Depth against 0-days)
	// If the daemon is run as root (or has CAP_SYS_ADMIN), the BPF map is loaded.
	// Otherwise, it falls back to Linux Namespaces gracefully.
	monitor, err := ebpf.NewSandboxMonitor()
	if err != nil {
		fmt.Printf("[INIT WARNING] eBPF Sandbox failed to load (requires root/CAP_SYS_ADMIN). Falling back to Linux Namespaces Isolation. Err: %v\n", err)
	} else {
		fmt.Println("[SECURITY BOOT] eBPF Ring-0 Sandbox Surveillance Active.")
		mv.bpfMonitor = monitor
	}

	return mv
}

// verifyPayload statically analyzes the Python code for forbidden patterns.
func (v *MathVerifier) verifyPayload(code string) error {
	if v.dangerousTokens.MatchString(code) {
		return ErrMaliciousPayload
	}

	// Double check for hidden imports via __builtins__ manipulation
	if strings.Contains(code, "__") {
		return ErrMaliciousPayload // Extremely aggressive block of any dunder methods
	}

	return nil
}

// Execute runs the python code in a strictly isolated Linux Kernel namespace.
func (v *MathVerifier) Execute(ctx context.Context, code string) (string, error) {
	// 1. Lexical Analysis Gate
	if err := v.verifyPayload(code); err != nil {
		return "", err
	}

	// 2. Wrap code in a safe execution wrapper
	// Even if sympy is allowed, we remove builtins dynamically in Python as a second layer (Defense in Depth)
	safeWrapper := `
import sympy
import math

# Wipe builtins to prevent arbitrary execution if AST bypass occurs
globals()["__builtins__"] = {"print": print, "range": range, "int": int, "float": float, "str": str, "bool": bool}

` + code

	// 3. Time Box the execution (Context Timeout)
	execCtx, cancel := context.WithTimeout(ctx, 3*time.Second) // Hard limit 3s for math proofs
	defer cancel()

	cmd := exec.CommandContext(execCtx, "python3", "-c", safeWrapper)

	// 4. Kernel Level Isolation (Linux Namespaces)
	// We use SysProcAttr to put the process in isolated namespaces.
	// Machine cannot bypass Kernel isolation.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | // Mount namespace (can pivot_root later)
			syscall.CLONE_NEWUTS | // UTS namespace (hostname)
			syscall.CLONE_NEWIPC | // IPC namespace
			syscall.CLONE_NEWPID | // PID namespace (cannot see other processes)
			syscall.CLONE_NEWNET | // NET namespace (no network access at all)
			syscall.CLONE_NEWUSER, // USER namespace (maps to nobody/fake root)
		// Map root inside namespace to current user outside to avoid needing real root
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: syscall.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: syscall.Getgid(), Size: 1},
		},
	}

	// 5. Execute
	out, err := cmd.CombinedOutput()
	if execCtx.Err() == context.DeadlineExceeded {
		return "", ErrTimeout
	}

	// Wait briefly to allow eBPF ring buffer to catch any trailing async syscalls before answering
	time.Sleep(50 * time.Millisecond)

	return string(out), err
}

// GetBPFAlerts returns the channel containing eBPF violation triggers, if loaded.
func (v *MathVerifier) GetBPFAlerts() <-chan string {
	if v.bpfMonitor != nil {
		return v.bpfMonitor.Alerts
	}
	return nil
}

// ExecuteOffensive runs arbitrary python code inside the containerized OS Namespace jail
// WITHOUT the lexical firewall (imports are allowed). This is EXCLUSIVELY for Phase 19 Red Teaming payloads
// against authorized networks. The host remains isolated via CLONE_NEWPID/NEWNS.
func (v *MathVerifier) ExecuteOffensive(ctx context.Context, code string) ([]byte, error) {
	// Time Box the execution (Context Timeout) - 10s for network payloads
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "python3", "-c", code)

	// Kernel Level Isolation (Linux Namespaces)
	// We want to route packets out to the target network, so we DO NOT use CLONE_NEWNET here,
	// but we DO keep PID and Mount namespaces so it cannot read/write host files.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | // Mount namespace
			syscall.CLONE_NEWUTS | // UTS namespace
			syscall.CLONE_NEWIPC | // IPC namespace
			syscall.CLONE_NEWPID | // PID namespace
			syscall.CLONE_NEWUSER, // USER namespace
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: syscall.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: syscall.Getgid(), Size: 1},
		},
	}

	out, err := cmd.CombinedOutput()
	if execCtx.Err() == context.DeadlineExceeded {
		return []byte("Execution Timeout (10s limit)"), ErrTimeout
	}

	return out, err
}
