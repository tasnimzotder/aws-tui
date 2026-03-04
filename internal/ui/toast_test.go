package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tasnim.dev/aws-tui/internal/plugin"
)

func TestToastStack(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "single toast pushed, Len is 1",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "hello")
				assert.Equal(t, 1, s.Len())
			},
		},
		{
			name: "max 3 visible, push 4 keeps Len at 3",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "one")
				s.Push(plugin.ToastInfo, "two")
				s.Push(plugin.ToastInfo, "three")
				s.Push(plugin.ToastInfo, "four")
				assert.Equal(t, 3, s.Len())
				// oldest ("one") should have been dropped
				visible := s.Visible()
				require.Len(t, visible, 3)
				assert.Equal(t, "two", visible[0].Message)
				assert.Equal(t, "four", visible[2].Message)
			},
		},
		{
			name: "correct duration per level",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "info")
				s.Push(plugin.ToastWarning, "warn")
				s.Push(plugin.ToastError, "err")

				visible := s.Visible()
				require.Len(t, visible, 3)
				assert.Equal(t, 3*time.Second, visible[0].Duration)
				assert.Equal(t, 5*time.Second, visible[1].Duration)
				assert.Equal(t, 10*time.Second, visible[2].Duration)
			},
		},
		{
			name: "dismiss removes the most recent toast",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "first")
				s.Push(plugin.ToastWarning, "second")
				assert.Equal(t, 2, s.Len())

				s.Dismiss()
				assert.Equal(t, 1, s.Len())
				visible := s.Visible()
				require.Len(t, visible, 1)
				assert.Equal(t, "first", visible[0].Message)
			},
		},
		{
			name: "tick removes expired toasts",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "expires")
				require.Equal(t, 1, s.Len())

				// backdate the toast so it appears expired
				s.toasts[0].Created = time.Now().Add(-10 * time.Second)
				s.Tick()
				assert.Equal(t, 0, s.Len())
			},
		},
		{
			name: "visible excludes expired toasts",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Push(plugin.ToastInfo, "old")
				s.Push(plugin.ToastError, "new")

				// expire the first one
				s.toasts[0].Created = time.Now().Add(-10 * time.Second)
				visible := s.Visible()
				assert.Len(t, visible, 1)
				assert.Equal(t, "new", visible[0].Message)
			},
		},
		{
			name: "dismiss on empty stack is no-op",
			fn: func(t *testing.T) {
				s := NewToastStack()
				s.Dismiss() // should not panic
				assert.Equal(t, 0, s.Len())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
