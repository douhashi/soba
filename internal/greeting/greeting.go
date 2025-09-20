package greeting

import "fmt"

// Hello returns a greeting message in English
func Hello(name string) string {
	if name == "" {
		name = "Guest"
	}
	return fmt.Sprintf("Hello, %s!", name)
}

// JapaneseGreeting returns a greeting message in Japanese
func JapaneseGreeting(name string) string {
	if name == "" {
		name = "ゲスト"
	}
	return fmt.Sprintf("こんにちは、%sさん！", name)
}