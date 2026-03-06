package reversing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m4rba4s/Nexus-LLM/internal/autonomy"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

// BinaryAnalyzer coordinates parsing ELF/PE files via R2 and passing them to the LLM
// to synthesize active defense rules like eBPF probes.
type BinaryAnalyzer struct {
	r2       *R2Wrapper
	provider core.Provider
	model    string
}

func NewBinaryAnalyzer(provider core.Provider, model string) *BinaryAnalyzer {
	return &BinaryAnalyzer{
		r2:       NewR2Wrapper(),
		provider: provider,
		model:    model,
	}
}

// AnalyzeAndSynthesize breaks down a binary target and asks the LLM to write a defense plan.
func (ba *BinaryAnalyzer) AnalyzeAndSynthesize(ctx context.Context, binaryPath string) (*autonomy.DefensePlan, error) {
	if ba.provider == nil {
		return nil, fmt.Errorf("LLM provider missing")
	}

	// 1. Run Headless Radare2 footprint
	dump, err := ba.r2.AnalyzeMainFunction(ctx, binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to disassemble binary %s: %w", binaryPath, err)
	}

	// 2. Format the raw assembly for the LLM context limits
	// We want to avoid dumping 10MB of text. We just dump the immediate ops.
	var asmBuilder strings.Builder
	for _, op := range dump.Ops {
		asmBuilder.WriteString(fmt.Sprintf("0x%x: %s\n", op.Offset, op.Disasm))
	}

	systemPrompt := `You are NexusLLM's Autonomous Reverse Engineering Core.
You will receive a Radare2 disassembly dump (JSON format) from an unknown binary's 'main' function.
Analyze the Assembly instructions (e.g., x86_64 SysV ABI). Identify if the binary uses suspicious
syscalls like 'execve' (0x3b), 'socket' (0x29) or drops executable files.

Generate a C code block for an eBPF Tracepoint probe designed to BLOCK the specific behavior you found.
Return ONLY valid JSON matching this structure:

{
  "SyscallTarget": "sys_enter_execve",
  "Confidence": "95%",
  "DefenseType": "ebpf",
  "BPFCode": "#include <vmlinux.h>\n..."
}

Do NOT wrap the JSON in markdown blocks. Provide valid JSON.`

	userPrompt := fmt.Sprintf("Target Binary Disassembly (main):\n%s", asmBuilder.String())

	req := &core.CompletionRequest{
		Model: ba.model,
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: systemPrompt},
			{Role: core.RoleUser, Content: userPrompt},
		},
	}

	// 3. Ask Claude/Gemini to reverse the payload
	resp, err := ba.provider.CreateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm synthesis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM empty response")
	}

	content := resp.Choices[0].Message.Content

	// Fast sanitize
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	content = strings.TrimSpace(content)

	var plan autonomy.DefensePlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("JSON parse error for DefensePlan: %w\nContent: %s", err, content)
	}

	return &plan, nil
}
