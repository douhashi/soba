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
			name:     "挨拶（名前あり）",
			input:    "World",
			expected: "Hello, World!",
		},
		{
			name:     "挨拶（空文字）",
			input:    "",
			expected: "Hello, Guest!",
		},
		{
			name:     "挨拶（日本語）",
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
			name:     "日本語挨拶（名前あり）",
			input:    "太郎",
			expected: "こんにちは、太郎さん！",
		},
		{
			name:     "日本語挨拶（空文字）",
			input:    "",
			expected: "こんにちは、ゲストさん！",
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
