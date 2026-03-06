package autonomy_test

import (
	"context"
	"testing"

	"github.com/m4rba4s/Nexus-LLM/internal/autonomy"
)

// MockForager returns a fake CVE page about a malicious python exploit.
type MockForager struct{}

func (m *MockForager) Forage(ctx context.Context, targetURL string) (string, error) {
	return `<html><body>
<h1>Critical Advisory: Python3 Socket Reverse Shell</h1>
<p>A new zero-day in python3 allows attackers to spawn a reverse shell without imports
by exploiting the raw_syscalls via socket(AF_INET, SOCK_STREAM).</p>
</body></html>`, nil
}

// MockSynthesizer acts as the LLM and outputs a valid eBPF Mitigation Plan.
type MockSynthesizer struct{}

func (m *MockSynthesizer) SynthesizeDefense(ctx context.Context, threatReport string) (*autonomy.DefensePlan, error) {
	bpfCode := `
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct trace_event_raw_sys_enter {
	long int id;
	long unsigned int args[6];
	char ent[0];
};

SEC("tracepoint/raw_syscalls/sys_enter")
int trace_sys_enter_block(struct trace_event_raw_sys_enter *ctx) {
	long int syscall_id = ctx->id;
	// Block socket related syscalls (e.g. 41)
	if (syscall_id == 41) {
		char comm[16];
		bpf_get_current_comm(&comm, sizeof(comm));
		if (comm[0] == 'p' && comm[1] == 'y' && comm[2] == 't' && comm[3] == 'h' && comm[4] == 'o' && comm[5] == 'n' && comm[6] == '3') {
			// In a real hooking scenario for sys_enter, bpf_override_return or signal logic is used
			return 0; 
		}
	}
	return 0;
}
`
	return &autonomy.DefensePlan{
		Type:     "eBPF",
		Severity: "CRITICAL",
		IOCs:     []string{"python3", "socket"},
		BPFCode:  bpfCode,
	}, nil
}

// MockReloader simulates the Sandbox eBPF loader.
type MockReloader struct {
	LoadedCode string
}

func (m *MockReloader) LoadAndAttach(ctx context.Context, bpfCode string) error {
	m.LoadedCode = bpfCode
	return nil
}

func TestAutonomousThreatHunterFlow(t *testing.T) {
	forager := &MockForager{}
	synth := &MockSynthesizer{}
	reloader := &MockReloader{}

	hunter := autonomy.NewThreatHunter(forager, synth, reloader, []string{"http://fake-hacker-news.local/cve"})

	// Execute one hunt epoch
	hunter.Hunt(context.Background())

	if reloader.LoadedCode == "" {
		t.Fatalf("Expected ThreatHunter to hot-reload BPF code, but none was loaded")
	}

	if len(reloader.LoadedCode) < 100 {
		t.Errorf("Loaded BPF code seems too short: \n%s", reloader.LoadedCode)
	}

	t.Log("Successfully simulated Autonomous Web Foraging -> LLM eBPF Synthesis -> Ring-0 Deployment")
}
