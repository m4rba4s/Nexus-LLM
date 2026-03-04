package biology

import (
	"log"
	"os"
	"time"
)

// Cell indicates a self-contained biological process unit (NexusLLM daemon).
type Cell struct {
	NervousSystem *NervousSystem
	Mitochondria  *Mitochondria
	Encoder       *GeneticEncoder
}

// Apoptosis is strictly a cleanup and self-destruct mechanism.
// It removes PIDs, closes sockets, and terminates the OS process (os.Exit).
// Used to prevent rogue Swarm nodes (Monsters) from living forever.
func ExecuteApoptosis(cell *Cell, reason string) {
	log.Printf("[BIOLOGY-FATAL] APOPTOSIS TRIGGERED: %s", reason)

	// 1. Drain energy to paralyze the cell
	if cell.Mitochondria != nil {
		cell.Mitochondria.Shutdown()
		log.Println(" |-> Mitochondria shutdown. Cellular respiration halted.")
	}

	// 2. Erase Memory buffers (Metaphorical DNA Degradation)
	// In Go, the garbage collector handles this, but we explicitly nil references.
	if cell.Encoder != nil {
		cell.Encoder = nil
		log.Println(" |-> Encoders destroyed. Genetic translation halted.")
	}

	// 3. Delete OS trails (PID files, temporary scratchpads)
	// e.g., os.Remove("/tmp/nexus_node.pid")
	log.Println(" |-> Scrubbing process artifacts.")

	// Simulate agonizing 2-second death to ensure async cleanup processes finish
	time.Sleep(2 * time.Second)

	log.Println("[BIOLOGY-FATAL] Biological substrate decomposed. Terminating process.")
	os.Exit(0)
}
