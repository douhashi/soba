// Package config provides configuration loading and management functionality.
package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// DisplayConfig 設定内容をマスキングしてYAML形式で返す
func DisplayConfig(cfg *Config) (string, error) {
	if cfg == nil {
		return "", errors.New("config is nil")
	}

	// センシティブな情報をマスキング
	masked := MaskSensitiveConfig(cfg)

	// YAMLにマーシャリング
	data, err := yaml.Marshal(masked)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// MaskSensitiveConfig センシティブな情報をマスキングした設定のコピーを返す
func MaskSensitiveConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	// ディープコピーを作成
	masked := *cfg

	// GitHubトークンをマスク（空でもマスク）
	masked.GitHub.Token = "***MASKED***"

	// Slack WebhookURLをマスク（空でもマスク）
	masked.Slack.WebhookURL = "***MASKED***"

	return &masked
}
