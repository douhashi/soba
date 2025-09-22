package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/domain"
)

func TestGetCurrentPhaseFromLabels(t *testing.T) {
	tests := []struct {
		name          string
		labels        []string
		expectedPhase domain.Phase
		expectedError bool
		errorContains string
	}{
		{
			name:          "soba:todoラベルがある場合はqueueフェーズ",
			labels:        []string{"soba:todo", "bug", "enhancement"},
			expectedPhase: domain.PhaseQueue,
		},
		{
			name:          "soba:queuedラベルがある場合はplanフェーズ",
			labels:        []string{"soba:queued", "bug"},
			expectedPhase: domain.PhasePlan,
		},
		{
			name:          "soba:planningラベルがある場合はplanフェーズ（実行中）",
			labels:        []string{"soba:planning", "enhancement"},
			expectedPhase: domain.PhasePlan,
		},
		{
			name:          "soba:readyラベルがある場合はimplementフェーズ",
			labels:        []string{"soba:ready", "feature"},
			expectedPhase: domain.PhaseImplement,
		},
		{
			name:          "soba:doingラベルがある場合はimplementフェーズ（実行中）",
			labels:        []string{"soba:doing", "bug"},
			expectedPhase: domain.PhaseImplement,
		},
		{
			name:          "soba:review-requestedラベルがある場合はreviewフェーズ",
			labels:        []string{"soba:review-requested", "enhancement"},
			expectedPhase: domain.PhaseReview,
		},
		{
			name:          "soba:reviewingラベルがある場合はreviewフェーズ（実行中）",
			labels:        []string{"soba:reviewing"},
			expectedPhase: domain.PhaseReview,
		},
		{
			name:          "soba:doneラベルがある場合はreviewフェーズ（完了）",
			labels:        []string{"soba:done", "enhancement"},
			expectedPhase: domain.PhaseReview,
		},
		{
			name:          "soba:requires-changesラベルがある場合はreviseフェーズ",
			labels:        []string{"soba:requires-changes"},
			expectedPhase: domain.PhaseRevise,
		},
		{
			name:          "soba:revisingラベルがある場合はreviseフェーズ（実行中）",
			labels:        []string{"soba:revising", "bug"},
			expectedPhase: domain.PhaseRevise,
		},
		{
			name:          "soba:lgtmラベルは無視される（他のsobaラベルがある場合）",
			labels:        []string{"soba:ready", "soba:lgtm", "feature"},
			expectedPhase: domain.PhaseImplement,
		},
		{
			name:          "sobaラベルがない場合はエラー",
			labels:        []string{"bug", "enhancement"},
			expectedError: true,
			errorContains: "no soba label found",
		},
		{
			name:          "複数のsobaラベルがある場合はエラー（LGTMを除く）",
			labels:        []string{"soba:todo", "soba:ready", "bug"},
			expectedError: true,
			errorContains: "multiple soba labels found",
		},
		{
			name:          "不明なsobaラベルがある場合はエラー",
			labels:        []string{"soba:unknown", "bug"},
			expectedError: true,
			errorContains: "unknown soba label: soba:unknown",
		},
		{
			name:          "空のラベルリストの場合はエラー",
			labels:        []string{},
			expectedError: true,
			errorContains: "no soba label found",
		},
		{
			name:          "soba:lgtmのみの場合はエラー",
			labels:        []string{"soba:lgtm"},
			expectedError: true,
			errorContains: "no soba label found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, err := domain.GetCurrentPhaseFromLabels(tt.labels)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPhase, phase)
			}
		})
	}
}
