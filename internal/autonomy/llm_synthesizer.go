package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

// ExecutorSynthesizer uses an LLM Provider (Opus/Gemini) to convert raw threat text into eBPF defenses.
type ExecutorSynthesizer struct {
	provider core.Provider
	model    string
}

func NewExecutorSynthesizer(provider core.Provider, model string) *ExecutorSynthesizer {
	return &ExecutorSynthesizer{provider: provider, model: model}
}

func (s *ExecutorSynthesizer) SynthesizeDefense(ctx context.Context, threatReport string) (*DefensePlan, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("LLM provider is missing")
	}

	systemPrompt := `You are NexusLLM's Autonomous Threat Intelligence & Defense Synthesis module.
Analyze the following threat report. Identify the core exploit mechanism (e.g., unauthorized bind/connect, specific syscall).
Generate a Mitigation Plan using C code for an eBPF probe that hooks the relevant syscall to drop malicious traffic or execution.

Your response MUST be exclusively a raw JSON object matching the following structure entirely:

{
  "Type": "eBPF",
  "BPFCode": "#include <linux/bpf.h>\n...",
  "IOCs": ["ip1", "hash1"],
  "Severity": "CRITICAL"
}

Do NOT wrap the JSON in markdown formatting. Provide valid JSON. DO NOT INCLUDE EXPLANATIONS outside of the JSON.`

	userPrompt := fmt.Sprintf("Analyze this intercept report:\n%s", threatReport)

	req := &core.CompletionRequest{
		Model: s.model,
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: systemPrompt},
			{Role: core.RoleUser, Content: userPrompt},
		},
	}

	resp, err := s.provider.CreateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content

	// Cleanup markdown tags if model ignored instructions
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	content = strings.TrimSpace(content)

	var plan DefensePlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON DefensePlan: %w\nContent: %s", err, content)
	}

	return &plan, nil
}
