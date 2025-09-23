package greeting

import (
	"testing"
)

func TestHello(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Greeting with name",
			input:    "World",
			expected: "Hello, World!",
		},
		{
			name:     "Greeting with empty string",
			input:    "",
			expected: "Hello, Guest!",
		},
		{
			name:     "Greeting with Japanese characters",
			input:    "世界",
			expected: "Hello, 世界!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Hello(tt.input)
			if result != tt.expected {
				t.Errorf("Hello(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestJapaneseGreeting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Japanese greeting with name",
			input:    "Taro",
			expected: "Hello, Taro!",
		},
		{
			name:     "Japanese greeting with empty string",
			input:    "",
			expected: "Hello, Guest!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JapaneseGreeting(tt.input)
			if result != tt.expected {
				t.Errorf("JapaneseGreeting(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
