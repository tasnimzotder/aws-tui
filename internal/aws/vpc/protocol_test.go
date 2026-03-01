package vpc

import "testing"

func TestNormalizeProtocol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"-1", "All"},
		{"6", "TCP"},
		{"17", "UDP"},
		{"1", "ICMP"},
		{"47", "47"},   // passthrough for unknown
		{"58", "58"},   // ICMPv6 passthrough
		{"", ""},       // empty passthrough
		{"tcp", "tcp"}, // already named, passthrough
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeProtocol(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeProtocol(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
