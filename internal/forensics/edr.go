package forensics

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EDRMonitor represents the Phase 22 Fileless Threat Detection engine.
type EDRMonitor struct {
	scanner     *Scanner
	synthesizer *YARASynthesizer
	targets     []string // list of process names to watch, e.g. ["python3", "bash"]
}

func NewEDRMonitor(ys *YARASynthesizer, targets []string) *EDRMonitor {
	return &EDRMonitor{
		scanner:     NewScanner(),
		synthesizer: ys,
		targets:     targets,
	}
}

// Start begins continuous Ring-3 memory surveillance.
func (e *EDRMonitor) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[EDR] Activating Fileless Memory Scanner. Polling interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("[EDR] Shutting down continuous memory forensics.")
			return
		case <-ticker.C:
			e.scanHost(ctx)
		}
	}
}

func (e *EDRMonitor) scanHost(ctx context.Context) {
	pids, err := e.findTargetPIDs()
	if err != nil {
		log.Printf("[EDR-WARN] Failed to enumerating running PIDs: %v", err)
		return
	}

	for _, pid := range pids {
		anomalies, err := e.scanner.FindAnomalousRegions(pid)
		if err != nil {
			continue // Process likely died or permission denied
		}

		for _, anomaly := range anomalies {
			log.Printf("[EDR-ALERT] Detected Anonymous RWX Memory Anomaly! PID: %d, Range: 0x%x - 0x%x", pid, anomaly.StartAddr, anomaly.EndAddr)

			// 1. Dump RAM securely
			dump, err := e.scanner.DumpMemory(pid, anomaly)
			if err != nil {
				log.Printf("[EDR-ERROR] Ptrace memory dump failed: %v", err)
				continue
			}

			// 2. Format specifically for LLM Token window
			// Max ~16KB of hex to keep context clean and cheap for Opus/Gemini
			hexContext := FormatDumpToHex(dump, 16384)

			// 3. Autonomous YARA Synthesis
			log.Printf("[EDR] Dispatching raw 0x%x bytecode to NexusLLM for Reverse Engineering and YARA synthesis...", anomaly.StartAddr)

			plan, err := e.synthesizer.GenerateSignature(ctx, pid, anomaly, hexContext)
			if err != nil {
				log.Printf("[EDR-ERROR] YARA LLM Synthesis failed: %v", err)
				continue
			}

			if plan.DefenseType == "yara" {
				log.Printf("[EDR-MITIGATION] Successfully generated synthetic YARA Rule with %s confidence. SIGKILLing infected PID: %d", plan.Confidence, pid)

				// 4. Terminate injected process
				proc, err := os.FindProcess(pid)
				if err == nil {
					_ = proc.Kill()
					log.Printf("[EDR-SUCCESS] Fileless Threat Neutralized. Process %d terminated.", pid)
					log.Printf("[EDR-YARA-RULE-EXPORT] \n%s\n", plan.BPFCode)
				}

				// Optional Future Step: Broadcast this YARA rule via the Phase 21 P2P Swarm Gossip!
			}
		}
	}
}

// findTargetPIDs enumerates /proc to find PIDs matching e.targets.
func (e *EDRMonitor) findTargetPIDs() ([]int, error) {
	var matchingPIDs []int

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a PID folder
		}

		exePath, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue // Access denied or process dead
		}

		exeName := filepath.Base(exePath)
		for _, target := range e.targets {
			if strings.Contains(exeName, target) {
				matchingPIDs = append(matchingPIDs, pid)
				break
			}
		}
	}

	return matchingPIDs, nil
}
