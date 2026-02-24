package utils

import "testing"

func TestShortName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"arn:aws:ecs:us-east-1:123456:task-definition/my-task:1", "my-task:1"},
		{"arn:aws:ecs:us-east-1:123456:task/my-cluster/abc123", "abc123"},
		{"plain-string", "plain-string"},
		{"single/segment", "segment"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ShortName(tt.input)
		if got != tt.want {
			t.Errorf("ShortName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSecondToLast(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/my-tg/abc123", "my-tg"},
		{"a/b/c", "b"},
		{"a/b", "a"},
		{"no-slash", "no-slash"},
		{"", ""},
	}

	for _, tt := range tests {
		got := SecondToLast(tt.input)
		if got != tt.want {
			t.Errorf("SecondToLast(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
