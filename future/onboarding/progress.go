package onboarding

import (
	"errors"
	"fmt"
	"time"
)

// CurriculumVersion is the checklist curriculum this build understands.
const CurriculumVersion = 1

// Step is one item of the welcome checklist.
type Step uint8

const (
	Terms Step = iota
	Credit
	Listing
	Partner
	Trade
	StepCount
)

var stepKeys = [StepCount]string{"terms", "credit", "listing", "partner", "trade"}

// Key returns the stable machine name of the step.
func (s Step) Key() string {
	if s >= StepCount {
		return ""
	}
	return stepKeys[s]
}

// ParseStep converts a stable key into a Step.
func ParseStep(key string) (Step, error) {
	for i, k := range stepKeys {
		if k == key {
			return Step(i), nil
		}
	}
	return 0, ErrInvalidStep
}

// Errors returned by the package. Callers may compare with errors.Is.
var (
	ErrInvalidStep       = errors.New("onboarding: invalid step")
	ErrInvalidState      = errors.New("onboarding: invalid tutorial state")
	ErrInvalidAction     = errors.New("onboarding: invalid action")
	ErrStaleCurriculum   = errors.New("onboarding: stored progress belongs to a newer curriculum")
	ErrConflict          = errors.New("onboarding: concurrent update, retry")
	ErrForbidden         = errors.New("onboarding: not permitted")
	ErrInvalidIdentifier = errors.New("onboarding: invalid network or member identifier")
)

const allSteps uint8 = 1<<StepCount - 1

// Progress is the non-authoritative learning state of one member for one
// curriculum version. Revision is a compare-and-set counter owned by the store.
type Progress struct {
	Version   int
	Completed uint8
	Skipped   uint8
	Dismissed bool
	// Share records the member's own consent to be counted in the network's
	// anonymous aggregate completion counters. Default false.
	Share     bool
	Revision  int64
	UpdatedAt time.Time
}

// Validate rejects states that could not have been produced by Apply. It is
// run on every load as well as every write.
func (p Progress) Validate() error {
	switch {
	case p.Version < 1:
		return fmt.Errorf("%w: version %d", ErrInvalidState, p.Version)
	case p.Completed&^allSteps != 0, p.Skipped&^allSteps != 0:
		return fmt.Errorf("%w: mask out of range", ErrInvalidState)
	case p.Completed&p.Skipped != 0:
		return fmt.Errorf("%w: step both completed and skipped", ErrInvalidState)
	case p.Revision < 0:
		return fmt.Errorf("%w: negative revision", ErrInvalidState)
	}
	return nil
}

// ActionKind enumerates checklist transitions.
type ActionKind uint8

const (
	ActComplete ActionKind = iota + 1 // mark a step as learned/done
	ActSkip                           // skip one step (resumable)
	ActResume                         // un-skip one step
	ActDismiss                        // skip the whole checklist (resumable)
	ActRestore                        // bring back a dismissed checklist
	ActShare                          // consent to anonymous aggregate counting
	ActUnshare                        // withdraw that consent
)

// Action is a requested transition. Step is used only by step actions.
type Action struct {
	Kind ActionKind
	Step Step
}

func (a Action) needsStep() bool {
	return a.Kind == ActComplete || a.Kind == ActSkip || a.Kind == ActResume
}

// Complete marks step done. It is idempotent and clears a skip on that step.
func (p Progress) Complete(step Step) (Progress, error) {
	return p.Apply(Action{ActComplete, step})
}

// Apply is a pure, idempotent transition. Re-applying an action never changes
// the result a second time, and actions on different steps commute, so two
// stale browser tabs cannot lose each other's completed steps.
func (p Progress) Apply(a Action) (Progress, error) {
	if err := p.Validate(); err != nil {
		return p, err
	}
	if a.needsStep() && a.Step >= StepCount {
		return p, ErrInvalidStep
	}
	bit := uint8(1) << a.Step
	switch a.Kind {
	case ActComplete:
		p.Completed |= bit
		p.Skipped &^= bit
	case ActSkip:
		if p.Completed&bit == 0 { // a finished step cannot be un-finished by skipping
			p.Skipped |= bit
		}
	case ActResume:
		p.Skipped &^= bit
	case ActDismiss:
		p.Dismissed = true
	case ActRestore:
		p.Dismissed = false
		p.Skipped = 0
	case ActShare:
		p.Share = true
	case ActUnshare:
		p.Share = false
	default:
		return p, ErrInvalidAction
	}
	return p, nil
}

// Next returns the first step that is neither completed nor skipped. It
// returns false when the checklist is dismissed or nothing is outstanding.
func (p Progress) Next() (Step, bool) {
	if p.Dismissed {
		return 0, false
	}
	for s := Terms; s < StepCount; s++ {
		if (p.Completed|p.Skipped)&(1<<s) == 0 {
			return s, true
		}
	}
	return 0, false
}

// Done reports whether every step is completed.
func (p Progress) Done() bool { return p.Completed == allSteps }

// IsCompleted reports whether step is completed.
func (p Progress) IsCompleted(s Step) bool { return s < StepCount && p.Completed&(1<<s) != 0 }

// IsSkipped reports whether step is skipped.
func (p Progress) IsSkipped(s Step) bool { return s < StepCount && p.Skipped&(1<<s) != 0 }

// newlyCompleted lists steps completed in next but not in prev.
func newlyCompleted(prev, next Progress) []Step {
	var out []Step
	for s := Terms; s < StepCount; s++ {
		if next.Completed&(1<<s) != 0 && prev.Completed&(1<<s) == 0 {
			out = append(out, s)
		}
	}
	return out
}
