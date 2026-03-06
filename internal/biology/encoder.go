package biology

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/autonomy"
)

// GeneticEncoder acts as RNA, translating, packaging, and mutating payloads.
type GeneticEncoder struct {
	entropySource *rand.Rand
}

func NewGeneticEncoder() *GeneticEncoder {
	return &GeneticEncoder{
		entropySource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Encode compress and serializes a DefensePlan into "genetic" byte sequences.
func (ge *GeneticEncoder) Encode(plan *autonomy.DefensePlan) ([]byte, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to encode (translate) genetic data: %w", err)
	}

	// 1. Apply structural headers (Metaphorical 'Start Codon' ATG)
	header := []byte{0x41, 0x54, 0x47} // ATG

	// 2. Transcribe
	payload := append(header, data...)

	// 3. Apply Metaphorical Termination (Stop Codon)
	footer := []byte{0x54, 0x41, 0x47} // TAG
	payload = append(payload, footer...)

	return payload, nil
}

// Mutate introduces calculated entropy (polymorphism) to bypass static signatures.
// Currently it simulates AST/NOP-sled shifts in BPF payloads.
func (ge *GeneticEncoder) Mutate(plan *autonomy.DefensePlan) *autonomy.DefensePlan {
	if plan.BPFCode == "" {
		return plan
	}

	// Simple simulation of polymorphism: Inject randomized NOP sleds or comments
	mutationTypes := []string{"// Mutated: Shift Offset", "// Poly: Obfuscate Variable", "// Insert: Dead Code"}
	chaos := mutationTypes[ge.entropySource.Intn(len(mutationTypes))]

	// Prepend mutation metaphor to payload structure
	plan.BPFCode = fmt.Sprintf("%s\n%s", chaos, plan.BPFCode)

	return plan
}

// Decode extracts the DefensePlan, validating biological integrity (Start/Stop codons).
func (ge *GeneticEncoder) Decode(raw []byte) (*autonomy.DefensePlan, error) {
	if len(raw) < 6 {
		return nil, fmt.Errorf("sequence too short to contain genetic markers")
	}

	// Verify Start Codon
	if raw[0] != 0x41 || raw[1] != 0x54 || raw[2] != 0x47 {
		return nil, fmt.Errorf("invalid start codon (missing ATG)")
	}

	// Verify Stop Codon
	ln := len(raw)
	if raw[ln-3] != 0x54 || raw[ln-2] != 0x41 || raw[ln-1] != 0x47 {
		return nil, fmt.Errorf("invalid stop codon (missing TAG)")
	}

	// Extract core payload
	core := raw[3 : ln-3]

	var plan autonomy.DefensePlan
	if err := json.Unmarshal(core, &plan); err != nil {
		return nil, fmt.Errorf("failed to decode genetic payload: %w", err)
	}

	return &plan, nil
}
