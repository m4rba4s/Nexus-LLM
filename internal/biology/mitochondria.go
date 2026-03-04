package biology

import (
	"errors"
	"sync"
	"time"
)

// ErrATPDepleted implies the Node has exhausted its compute energy.
var ErrATPDepleted = errors.New("ATP (energy) depleted. Action denied.")

// Mitochondria acts as a Token Bucket rate limiter, managing "ATP" (Energy).
type Mitochondria struct {
	mu           sync.Mutex
	currentATP   int
	maxATP       int
	regenRate    int // ATP regenerated per tick
	tickDuration time.Duration
	stopCh       chan struct{}
}

// NewMitochondria creates an energy manager.
// Default: Max 100 ATP, regens 5 ATP every 10 seconds.
func NewMitochondria(max, rate int, tick time.Duration) *Mitochondria {
	m := &Mitochondria{
		currentATP:   max,
		maxATP:       max,
		regenRate:    rate,
		tickDuration: tick,
		stopCh:       make(chan struct{}),
	}
	m.startCycle()
	return m
}

// startCycle manages the Krebs cycle (ATP regeneration over time).
func (m *Mitochondria) startCycle() {
	go func() {
		ticker := time.NewTicker(m.tickDuration)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.mu.Lock()
				m.currentATP += m.regenRate
				if m.currentATP > m.maxATP {
					m.currentATP = m.maxATP
				}
				m.mu.Unlock()
			}
		}
	}()
}

// Consume attempts to spend ATP for a heavy computational task (e.g. LLM API call, EDR Scan).
func (m *Mitochondria) Consume(cost int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentATP >= cost {
		m.currentATP -= cost
		return nil
	}

	return ErrATPDepleted
}

// GetATP returns current energy level.
func (m *Mitochondria) GetATP() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentATP
}

// Shutdown halts cellular respiration.
func (m *Mitochondria) Shutdown() {
	close(m.stopCh)
}
