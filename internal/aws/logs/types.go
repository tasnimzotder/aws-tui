package logs

import "time"

// LogEvent represents a single CloudWatch log event.
type LogEvent struct {
	Timestamp time.Time
	Message   string
}
