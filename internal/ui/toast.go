package ui

import (
	"time"

	"tasnim.dev/aws-tui/internal/plugin"
)

const maxToasts = 3

// Toast represents a single notification.
type Toast struct {
	Level    plugin.ToastLevel
	Message  string
	Duration time.Duration
	Created  time.Time
}

// ToastStack manages a bounded slice of toasts.
type ToastStack struct {
	toasts []Toast
}

// NewToastStack returns an initialised ToastStack.
func NewToastStack() *ToastStack {
	return &ToastStack{}
}

// Push adds a toast with a duration determined by its level.
// If the stack already holds maxToasts, the oldest toast is dropped.
func (s *ToastStack) Push(level plugin.ToastLevel, msg string) {
	var dur time.Duration
	switch level {
	case plugin.ToastInfo:
		dur = 3 * time.Second
	case plugin.ToastWarning:
		dur = 5 * time.Second
	case plugin.ToastError:
		dur = 10 * time.Second
	default:
		dur = 3 * time.Second
	}

	s.toasts = append(s.toasts, Toast{
		Level:    level,
		Message:  msg,
		Duration: dur,
		Created:  time.Now(),
	})

	if len(s.toasts) > maxToasts {
		s.toasts = s.toasts[len(s.toasts)-maxToasts:]
	}
}

// Visible returns toasts that have not yet expired.
func (s *ToastStack) Visible() []Toast {
	now := time.Now()
	var out []Toast
	for _, t := range s.toasts {
		if t.Created.Add(t.Duration).After(now) {
			out = append(out, t)
		}
	}
	return out
}

// Tick removes expired toasts from the stack.
func (s *ToastStack) Tick() {
	now := time.Now()
	var kept []Toast
	for _, t := range s.toasts {
		if t.Created.Add(t.Duration).After(now) {
			kept = append(kept, t)
		}
	}
	s.toasts = kept
}

// Len returns the current number of toasts in the stack.
func (s *ToastStack) Len() int {
	return len(s.toasts)
}

// Dismiss removes the most recent (last) toast.
func (s *ToastStack) Dismiss() {
	if len(s.toasts) == 0 {
		return
	}
	s.toasts = s.toasts[:len(s.toasts)-1]
}
