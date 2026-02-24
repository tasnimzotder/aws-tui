package utils

import (
	"testing"
	"time"
)

func TestCurrency(t *testing.T) {
	tests := []struct {
		amount   float64
		currency string
		want     string
	}{
		{12.34, "USD", "$12.34"},
		{0.00, "USD", "$0.00"},
		{1234.56, "", "$1234.56"},
		{50.00, "EUR", "EUR 50.00"},
	}

	for _, tt := range tests {
		got := Currency(tt.amount, tt.currency)
		if got != tt.want {
			t.Errorf("Currency(%f, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}

func TestTimeOrDash(t *testing.T) {
	tests := []struct {
		name   string
		t      time.Time
		layout string
		want   string
	}{
		{"zero time", time.Time{}, DateTime, "—"},
		{"valid date", time.Date(2026, 2, 25, 14, 30, 0, 0, time.UTC), DateTime, "2026-02-25 14:30"},
		{"date only", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), DateOnly, "2026-01-01"},
		{"with seconds", time.Date(2026, 3, 15, 8, 45, 30, 0, time.UTC), DateTimeSec, "2026-03-15 08:45:30"},
		{"time only", time.Date(2026, 1, 1, 9, 5, 12, 0, time.UTC), TimeOnly, "09:05:12"},
	}

	for _, tt := range tests {
		got := TimeOrDash(tt.t, tt.layout)
		if got != tt.want {
			t.Errorf("TimeOrDash(%v, %q) = %q, want %q", tt.t, tt.layout, got, tt.want)
		}
	}
}
