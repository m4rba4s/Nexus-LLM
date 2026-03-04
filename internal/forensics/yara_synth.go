package forensics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourusername/gollm/internal/autonomy"
	"github.com/yourusername/gollm/internal/core"
)

// YARASynthesizer is responsible for translating raw RAM hex dumps into structured YARA rules.
type YARASynthesizer struct {
	provider core.Provider
	model    string
}

func NewYARASynthesizer(provider core.Provider, model string) *YARASynthesizer {
	return &YARASynthesizer{
		provider: provider,
		model:    model,
	}
}

// GenerateSignature takes raw bytes from an anomalous RWX memory region
// and orchestrates the LLM to write a high-fidelity YARA rule.
func (ys *YARASynthesizer) GenerateSignature(ctx context.Context, pid int, anomaly MemoryRegion, hexDump string) (*autonomy.DefensePlan, error) {
	if ys.provider == nil {
		return nil, fmt.Errorf("LLM provider missing")
	}

	systemPrompt := `You are NexusLLM's EDR (Endpoint Detection and Response) Memory Forensics AI.
You will receive a raw Hexadecimal Dump of a highly suspicious RWX (Read-Write-Execute) memory region
extracted dynamically from a live Linux process. 

Your objective is to:
1. Analyze the hex bytes. Look for standard x86_64/ARM prologues, known shellcode constants, or Reflective DLL markers.
2. Select a highly unique byte sequence (at least 8-16 bytes) to act as a signature.
3. Generate a strict, valid YARA rule that triggers on this exact malware/shellcode family.

Return ONLY valid JSON matching this structure:

{
  "SyscallTarget": "memory_injection",
  "Confidence": "99%",
  "DefenseType": "yara",
  "BPFCode": "rule Autonomous_Catch {\n  meta:\n    description = \"Auto-generated from RWX anomaly\"\n  strings:\n    $hex_sig = { 90 90 90 48 89 e5 ... }\n  condition:\n    any of them\n}"
}

Do NOT wrap the JSON in markdown blocks. Provide valid JSON. Encode the YARA rule as a string within 'BPFCode'.`

	userPrompt := fmt.Sprintf("Memory Target (PID: %d, Addr: 0x%x - 0x%x):\n\n%s",
		pid, anomaly.StartAddr, anomaly.EndAddr, hexDump)

	req := &core.CompletionRequest{
		Model: ys.model,
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: systemPrompt},
			{Role: core.RoleUser, Content: userPrompt},
		},
	}

	resp, err := ys.provider.CreateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm YARA synthesis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM empty response")
	}

	content := resp.Choices[0].Message.Content

	// Sanitize Markdown
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	content = strings.TrimSpace(content)

	var plan autonomy.DefensePlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("JSON parse error for DefensePlan (YARA): %w\nContent: %s", err, content)
	}

	return &plan, nil
}
