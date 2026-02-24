package utils

import (
	"fmt"
	"time"
)

const (
	DateOnly    = "2006-01-02"
	DateTime    = "2006-01-02 15:04"
	DateTimeSec = "2006-01-02 15:04:05"
	TimeOnly    = "15:04:05"
)

// Currency formats an amount with a currency symbol.
// USD (or empty) uses "$"; other currencies prefix with the code.
func Currency(amount float64, currency string) string {
	symbol := "$"
	if currency != "" && currency != "USD" {
		symbol = currency + " "
	}
	return fmt.Sprintf("%s%.2f", symbol, amount)
}

// TimeOrDash formats a time value using the given layout, or returns "—" if zero.
func TimeOrDash(t time.Time, layout string) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format(layout)
}
