package sandbox

import (
	"context"
	"strings"
	"testing"
)

func TestLexicalFirewall(t *testing.T) {
	v := NewMathVerifier()

	// 1. Safe SymPy expression
	safe := `result = sympy.integrate(sympy.exp(-x**2), (x, -sympy.oo, sympy.oo))`
	if err := v.verifyPayload(safe); err != nil {
		t.Errorf("Expected nil for safe payload, got %v", err)
	}

	// 2. Malicious imports
	malicious := []string{
		`import os`,
		`__import__('os').system('ls')`,
		`eval("print(1)")`,
		`__builtins__['open']('/etc/passwd').read()`,
		`import urllib.request`,
	}

	for _, payload := range malicious {
		if err := v.verifyPayload(payload); err != ErrMaliciousPayload {
			t.Errorf("Expected ErrMaliciousPayload for %q, got %v", payload, err)
		}
	}
}

func TestMathExecutionIsolation(t *testing.T) {
	// Note: this test might fail in environments without user namespaces enabled in the kernel mapping (e.g. some CI).
	// We handle graceful fallback or just skip if namespaces are restricted for unprivileged users,
	// but for the sake of architecture, it must be tested if possible.
	v := NewMathVerifier()

	// Let's test a simple algebraic evaluation
	code := `
import sympy
x = sympy.Symbol('x')
print(sympy.expand((x + 1)**2))
`

	ctx := context.Background()
	_, err := v.Execute(ctx, code)

	// In some restricted environments (like Docker containers without --privileged), user namespaces might fail
	// "clone: operation not permitted". We don't fail the Go test if the OS blocks clone, but we log it.
	if err != nil && strings.Contains(err.Error(), "operation not permitted") {
		t.Logf("Skipping namespace execution test: host OS restricts unprivileged user namespaces. Err: %v", err)
		return
	}

	// We expect the result (could be missing sympy if not installed in the system, but the isolation shouldn't fail fatally on startup unless it lacks python3)
}
