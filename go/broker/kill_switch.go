package broker

import "sync"

// KillSwitch is a thread-safe placeholder for the phase 7 kill switch.
type KillSwitch struct {
	mu      sync.Mutex
	engaged bool
}

// NewKillSwitch builds a disengaged kill switch.
func NewKillSwitch() *KillSwitch { return &KillSwitch{} }

// Engage stops new paper orders.
func (k *KillSwitch) Engage() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.engaged = true
}

// Disengage allows paper orders again.
func (k *KillSwitch) Disengage() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.engaged = false
}

// Engaged reports whether the switch is engaged.
func (k *KillSwitch) Engaged() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.engaged
}
