package service

import (
	"fmt"
	"time"
)

// Authentication throttle bounds preserve or tighten the existing guessing
// limits. Four attempts is the minimum that still grants two attempts after
// an expired lock restores half the allowance (SEC-006 / A3-02).
const (
	MinAuthThrottleAttempts           = 4
	DefaultAuthThrottleAttempts       = 5
	DefaultAuthThrottleGlobalAttempts = 100
	DefaultAuthThrottleLock           = 15 * time.Minute
	MaxAuthThrottleLock               = 24 * time.Hour
)

func configureAuthThrottle(cfg *Config) error {
	if cfg.AuthThrottleAttempts == 0 {
		cfg.AuthThrottleAttempts = DefaultAuthThrottleAttempts
	}
	if cfg.AuthThrottleGlobalAttempts == 0 {
		cfg.AuthThrottleGlobalAttempts = DefaultAuthThrottleGlobalAttempts
	}
	if cfg.AuthThrottleLock == 0 {
		cfg.AuthThrottleLock = DefaultAuthThrottleLock
	}
	if cfg.AuthThrottleAttempts < MinAuthThrottleAttempts || cfg.AuthThrottleAttempts > DefaultAuthThrottleAttempts {
		return fmt.Errorf("auth-throttle-attempts must be between %d and %d", MinAuthThrottleAttempts, DefaultAuthThrottleAttempts)
	}
	if cfg.AuthThrottleGlobalAttempts < cfg.AuthThrottleAttempts || cfg.AuthThrottleGlobalAttempts > DefaultAuthThrottleGlobalAttempts {
		return fmt.Errorf("auth-throttle-global-attempts must be between auth-throttle-attempts (%d) and %d", cfg.AuthThrottleAttempts, DefaultAuthThrottleGlobalAttempts)
	}
	if cfg.AuthThrottleLock < DefaultAuthThrottleLock || cfg.AuthThrottleLock > MaxAuthThrottleLock || cfg.AuthThrottleLock%time.Second != 0 {
		return fmt.Errorf("auth-throttle-lock must be whole seconds between %s and %s", DefaultAuthThrottleLock, MaxAuthThrottleLock)
	}
	return nil
}
