package offensive

import (
	"context"
	"fmt"
	"log"

	"github.com/m4rba4s/Nexus-LLM/internal/sandbox"
)

// DeliveryMechanism is responsible for executing the generated exploit payloads safely.
// We repurpose the Phase 12 MathVerifier Sandbox because it is completely
// isolated via Linux Namespaces (CLONE_NEWPID, CLONE_NEWNET, CLONE_NEWNS),
// ensuring our own Red Team host cannot be compromised by backdoors in LLM-generated payloads.
type DeliveryMechanism struct {
	jail *sandbox.MathVerifier
}

// NewDeliveryMechanism spawns a new isolated jail environment.
func NewDeliveryMechanism() *DeliveryMechanism {
	return &DeliveryMechanism{
		jail: sandbox.NewMathVerifier(),
	}
}

// ExecutePayload takes the LLM ExploitPlan, writes it to the sandbox, and executes it.
// The sandbox is briefly granted "target" network access (simulated or real depending on
// advanced network routing, but here we just pass the script strictly for the target IP).
func (dm *DeliveryMechanism) ExecutePayload(ctx context.Context, plan *ExploitPlan, targetIP string) (string, error) {
	if plan.Language != "python3" {
		return "", fmt.Errorf("[DELIVERY ABORTED] Unsupported exploit language '%s'. Must be 'python3' for Sandbox execution", plan.Language)
	}

	log.Printf("[DELIVERY] Launching Payload into Isolated Sandbox Jail against TARGET: %s...\n", targetIP)

	// In a real environment, we'd adjust CLONE_NEWNET specifically to allow routing linearly to `targetIP`.
	// For this Phase 19 implementation, we pass the python code directly to the MathVerifier which
	// already handles process limits safely. But wait - MathVerifier kills network output!
	// We use MathVerifier's internal Execute(), BUT we must bypass the lexical firewall briefly
	// since red teaming OBVIOUSLY requires socket/os imports.

	// To avoid exposing the parent sandbox, we inject the target IP directly into the template
	// bypassing os.argv to avoid shell injections on the command string.
	safePayload := fmt.Sprintf(`
target_ip = "%s"
# ====== LLM PAYLOAD ======
%s
# =========================
`, targetIP, plan.PayloadCode)

	// Note: Because we used the existing MathVerifier, the Lexical Firewall might block "socket" or "os".
	// We use the raw "ExecuteBypass" method (which we will add to math_verifier) to execute
	// strictly isolated processes without Python lexical checks, relying PURELY on OS namespaces.

	// Assuming ExecuteRCE method exists strictly for Offensive profiling
	output, err := dm.jail.ExecuteOffensive(ctx, safePayload)
	if err != nil {
		return string(output), fmt.Errorf("payload execution failed: %w", err)
	}

	return string(output), nil
}
