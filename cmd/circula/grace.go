package main

import "time"

// graceFor is the shutdown grace period: the configured shutdown-grace or the
// 10 s default (ARC-016).
func graceFor(cfg *Config) time.Duration {
	if cfg != nil && cfg.ShutdownGrace > 0 {
		return cfg.ShutdownGrace
	}
	return shutdownGracePeriod
}
