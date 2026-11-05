package strategy

// OverlayConfigResolver mirrors Java.
func OverlayRegimeGate(overlays map[string]any) *RegimeGateConfig {
	if !overlayEnabled(overlays, "regimeGate") {
		return nil
	}
	c := RegimeGateDefaults()
	return &c
}

func OverlaySentimentGate(overlays map[string]any) *SentimentGateConfig {
	if !overlayEnabled(overlays, "sentimentGate") {
		return nil
	}
	c := SentimentGateDefaults()
	return &c
}

func overlayEnabled(overlays map[string]any, key string) bool {
	if overlays == nil {
		return false
	}
	v, ok := overlays[key]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if m, ok := v.(map[string]any); ok {
		if e, ok := m["enabled"]; ok {
			if bb, ok := e.(bool); ok {
				return bb
			}
			return false
		}
		return len(m) > 0
	}
	return false
}
