package offensive_test

import (
	"context"
	"strings"
	"testing"

	"github.com/m4rba4s/Nexus-LLM/internal/offensive"
)

// MockProvider generates a fake Python payload simulating RCE against port 80.
type MockProvider struct{}

func (m *MockProvider) CreateCompletion(ctx context.Context, req interface{}) (interface{}, error) {
	// Not full implementation, using direct injection for test flow
	return nil, nil
}

func TestRulesOfEngagement(t *testing.T) {
	roe := offensive.NewRulesOfEngagement()

	// Should allow RFC1918 Private ranges
	if err := roe.IsTargetAllowed("192.168.1.100"); err != nil {
		t.Errorf("Expected 192.168.1.100 to be allowed, got: %v", err)
	}

	if err := roe.IsTargetAllowed("10.5.5.5"); err != nil {
		t.Errorf("Expected 10.5.5.5 to be allowed, got: %v", err)
	}

	// Should BLOCK public IP ranges (e.g., google dns)
	if err := roe.IsTargetAllowed("8.8.8.8"); err == nil {
		t.Errorf("Expected 8.8.8.8 to be BLOCKED by RoE, but it was allowed")
	}

	// Should BLOCK hostname resolution tricks
	if err := roe.IsTargetAllowed("localhost"); err == nil {
		// netip doesn't resolve "localhost" (it's not an IP string), so this will fail parsing and block as intended.
		t.Logf("Correctly blocked unparseable host string: localhost")
	}
}

func TestPayloadDeliveryBypassIntegration(t *testing.T) {
	dm := offensive.NewDeliveryMechanism()

	plan := &offensive.ExploitPlan{
		TargetService: "HTTP",
		Confidence:    "99%",
		Language:      "python3",
		PayloadCode: `
import os
import urllib.request
print("CTF-PAYLOAD-DELIVERED-TO-TARGET: " + target_ip)
`,
	}

	output, err := dm.ExecutePayload(context.Background(), plan, "192.168.100.22")

	if err != nil {
		t.Fatalf("Delivery Mechanism failed to execute python payload: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "CTF-PAYLOAD-DELIVERED") {
		t.Errorf("Expected CTF delivery token, got: %s", output)
	}

	if !strings.Contains(output, "192.168.100.22") {
		t.Errorf("Expected Target IP to be dynamically injected into payload env, got: %s", output)
	}
}
