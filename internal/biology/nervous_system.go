package biology

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
)

// PainThreshold defines how much system trauma a node can handle before triggering Apoptosis.
const PainThreshold = 100

// NervousSystem handles signals (pain receptors) and triggers defensive reflexes.
type NervousSystem struct {
	mu          sync.Mutex
	painLevel   int
	engine      *kinematics.Engine
	currentUser kinematics.UserIdentity
	apoptosisCh chan struct{}
}

// NewNervousSystem attaches to OS interrupts and the core Kinematics engine.
func NewNervousSystem(eng *kinematics.Engine, user kinematics.UserIdentity) *NervousSystem {
	ns := &NervousSystem{
		engine:      eng,
		currentUser: user,
		apoptosisCh: make(chan struct{}),
	}
	ns.registerReceptors()
	return ns
}

// registerReceptors binds to OS signals mimicking painful external stimuli.
func (ns *NervousSystem) registerReceptors() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)

	go func() {
		for sig := range sigs {
			log.Printf("[BIOLOGY-WARN] External stimulus detected: %v", sig)
			ns.InducePain(50) // External termination signals are highly painful
		}
	}()
}

// InducePain increments the trauma level. If it exceeds PainThreshold, the node dies.
func (ns *NervousSystem) InducePain(amount int) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.painLevel += amount

	// Reflex: High pain massively spikes the core Fatigue matrix (Phase 1)
	inputVec := kinematics.InputVector{
		Errors: float64(amount) / 100.0,
	}
	state := ns.engine.Update(inputVec, ns.currentUser)
	log.Printf("[BIOLOGY] Fatigue matrix reflex adjusted to: %.2f", state.Fatigue)

	if ns.painLevel >= PainThreshold {
		log.Printf("[BIOLOGY-CRITICAL] Pain threshold exceeded (%d/%d). Initiating cellular death.", ns.painLevel, PainThreshold)
		close(ns.apoptosisCh) // Trigger Apoptosis
	}
}

// WaitForDeath blocks until the system triggers self-destruction.
func (ns *NervousSystem) WaitForDeath() <-chan struct{} {
	return ns.apoptosisCh
}
