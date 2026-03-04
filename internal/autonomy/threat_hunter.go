package autonomy

import (
	"context"
	"log"
	"time"
)

// WebForager defines the interface for scraping threat intelligence.
type WebForager interface {
	Forage(ctx context.Context, targetURL string) (string, error)
}

// LLMSynthesizer analyzes raw HTML/text and generates eBPF rules or IOCs.
type LLMSynthesizer interface {
	SynthesizeDefense(ctx context.Context, threatReport string) (*DefensePlan, error)
}

// DefensePlan represents the parsed strategy from the LLM.
type DefensePlan struct {
	Type          string   `json:"Type,omitempty"`        // e.g., "eBPF", "YARA"
	DefenseType   string   `json:"DefenseType,omitempty"` // used in Phase 20
	SyscallTarget string   `json:"SyscallTarget,omitempty"`
	Confidence    string   `json:"Confidence,omitempty"`
	BPFCode       string   `json:"BPFCode"` // Raw C code intended for the Ring-0 probe
	IOCs          []string `json:"IOCs,omitempty"`
	Severity      string   `json:"Severity,omitempty"`
}

// SandboxHotReloader applies the generated defense into the kernel.
type SandboxHotReloader interface {
	LoadAndAttach(ctx context.Context, bpfCode string) error
}

// ThreatHunter orchestrates autonomous web scraping, analysis, and kernel patching.
type ThreatHunter struct {
	forager     WebForager
	synthesizer LLMSynthesizer
	reloader    SandboxHotReloader
	targets     []string
}

// NewThreatHunter initializes a new autonomous threat hunter.
func NewThreatHunter(forager WebForager, synthesizer LLMSynthesizer, reloader SandboxHotReloader, targets []string) *ThreatHunter {
	return &ThreatHunter{
		forager:     forager,
		synthesizer: synthesizer,
		reloader:    reloader,
		targets:     targets,
	}
}

// Start runs the threat hunter loop on a given ticker interval.
func (th *ThreatHunter) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[TR-HUNTER] Started autonomous threat hunting loop (Interval: %v)", interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("[TR-HUNTER] Shutting down autonomous loop...")
			return
		case <-ticker.C:
			th.Hunt(ctx)
		}
	}
}

// Hunt executes a single run: scrape -> synthesize -> deploy.
func (th *ThreatHunter) Hunt(ctx context.Context) {
	log.Println("[TR-HUNTER] Initiating Threat Hunt sequence...")

	for _, target := range th.targets {
		// 1. Web Foraging
		log.Printf("[TR-HUNTER] Foraging intelligence from: %s", target)
		report, err := th.forager.Forage(ctx, target)
		if err != nil {
			log.Printf("[TR-HUNTER] WARN - Failed to forage %s: %v", target, err)
			continue
		}

		// 2. LLM Synthesis
		log.Printf("[TR-HUNTER] Synthesizing defense from report (len: %d bytes)...", len(report))
		plan, err := th.synthesizer.SynthesizeDefense(ctx, report)
		if err != nil {
			log.Printf("[TR-HUNTER] ERROR - LLM Synthesis failed: %v", err)
			continue
		}

		if plan == nil || plan.Type != "eBPF" {
			log.Println("[TR-HUNTER] No actionable eBPF defense plan detected. Skipping.")
			continue
		}

		// 3. Hot-Reload Sandbox
		log.Printf("[TR-HUNTER] Deploying synthesized eBPF Ring-0 mitigation (Severity: %s)...", plan.Severity)
		err = th.reloader.LoadAndAttach(ctx, plan.BPFCode)
		if err != nil {
			log.Printf("[TR-HUNTER] CRITICAL - Hot-Reload failed for generated eBPF code: %v", err)
			continue
		}

		log.Println("[TR-HUNTER] Successfully deployed autonomous defense module!")
	}
}
