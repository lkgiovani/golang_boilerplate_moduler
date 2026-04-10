package securityusecases

import "time"

// Policy defines thresholds and windows governing suspicious-activity auto-blocking.
type Policy struct {
	Window            time.Duration
	CriticalThreshold int64
	HighThreshold     int64
	BlockDuration     time.Duration
}

// DefaultPolicy returns the phase-06 defaults: 1h window, 3 CRITICAL → 24h block.
func DefaultPolicy() Policy {
	return Policy{
		Window:            time.Hour,
		CriticalThreshold: 3,
		HighThreshold:     10,
		BlockDuration:     24 * time.Hour,
	}
}
