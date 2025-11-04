package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSobaLabels(t *testing.T) {
	labels := GetSobaLabels()

	// 既存のラベルが含まれていることを確認
	expectedLabels := map[string]bool{
		"soba:todo":             false,
		"soba:todo:high":        false,
		"soba:todo:low":         false,
		"soba:queued":           false,
		"soba:planning":         false,
		"soba:ready":            false,
		"soba:doing":            false,
		"soba:review-requested": false,
		"soba:reviewing":        false,
		"soba:done":             false,
		"soba:requires-changes": false,
		"soba:revising":         false,
		"soba:lgtm":             false,
	}

	// 全てのラベルが定義されていることを確認
	assert.Equal(t, len(expectedLabels), len(labels), "ラベル数が一致しません")

	// 各ラベルが正しく定義されていることを確認
	for _, label := range labels {
		if _, exists := expectedLabels[label.Name]; exists {
			expectedLabels[label.Name] = true
			// ラベルの色とdescriptionが設定されていることを確認
			assert.NotEmpty(t, label.Color, "ラベル %s の色が設定されていません", label.Name)
			assert.NotEmpty(t, label.Description, "ラベル %s の説明が設定されていません", label.Name)
		}
	}

	// 全ての期待されるラベルが見つかったか確認
	for name, found := range expectedLabels {
		assert.True(t, found, "期待されるラベル %s が見つかりません", name)
	}
}

func TestGetSobaLabels_Priority(t *testing.T) {
	labels := GetSobaLabels()

	// 優先度ラベルが正しく定義されているか確認
	priorityLabels := map[string]struct {
		expectedColor string
		expectedDesc  string
	}{
		"soba:todo:high": {
			expectedColor: "d73a4a", // 赤系の色（高優先度）
			expectedDesc:  "High priority task",
		},
		"soba:todo:low": {
			expectedColor: "cfd3d7", // グレー系の色（低優先度）
			expectedDesc:  "Low priority task",
		},
	}

	for _, label := range labels {
		if expected, ok := priorityLabels[label.Name]; ok {
			assert.Equal(t, expected.expectedColor, label.Color,
				"ラベル %s の色が期待値と異なります", label.Name)
			assert.Equal(t, expected.expectedDesc, label.Description,
				"ラベル %s の説明が期待値と異なります", label.Name)
		}
	}
}