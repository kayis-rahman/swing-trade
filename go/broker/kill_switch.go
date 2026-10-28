package broker

// KillSwitch is a thread-safe placeholder for the phase 7 kill switch.
type KillSwitch struct {
	engaged bool
}

func NewKillSwitch() *KillSwitch { return &KillSwitch{} }
func (k *KillSwitch) Engage()    { k.engaged = true }
func (k *KillSwitch) Disengage() { k.engaged = false }
func (k *KillSwitch) Engaged() bool { return k.engaged }
