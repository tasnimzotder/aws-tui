package plugin

import "time"

// ResolveInterval returns the appropriate polling interval based on the
// current activity state. If IsActive is non-nil and returns true, the
// ActiveInterval is used; otherwise the IdleInterval is returned.
func ResolveInterval(cfg PollConfig) time.Duration {
	if cfg.IsActive != nil && cfg.IsActive() {
		return cfg.ActiveInterval
	}
	return cfg.IdleInterval
}
