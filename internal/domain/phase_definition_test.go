package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/domain"
)

func TestPhaseDefinitions(t *testing.T) {
	tests := []struct {
		name                string
		phaseName           string
		expectedTrigger     string
		expectedExecution   string
		expectedType        domain.PhaseExecutionType
		expectedPane        bool
		expectedWorktree    bool
		expectedCompletions map[string]bool // 完了ラベルの存在確認
	}{
		{
			name:              "queue フェーズが正しく定義されている",
			phaseName:         "queue",
			expectedTrigger:   domain.LabelTodo,
			expectedExecution: domain.LabelQueued,
			expectedType:      domain.ExecutionTypeLabelOnly,
			expectedPane:      false,
			expectedWorktree:  false,
			expectedCompletions: map[string]bool{
				domain.LabelQueued: true,
			},
		},
		{
			name:              "plan フェーズが正しく定義されている",
			phaseName:         "plan",
			expectedTrigger:   domain.LabelQueued,
			expectedExecution: domain.LabelPlanning,
			expectedType:      domain.ExecutionTypeCommand,
			expectedPane:      true,
			expectedWorktree:  true,
			expectedCompletions: map[string]bool{
				domain.LabelReady: true,
			},
		},
		{
			name:              "implement フェーズが正しく定義されている",
			phaseName:         "implement",
			expectedTrigger:   domain.LabelReady,
			expectedExecution: domain.LabelDoing,
			expectedType:      domain.ExecutionTypeCommand,
			expectedPane:      true,
			expectedWorktree:  true,
			expectedCompletions: map[string]bool{
				domain.LabelReviewRequested: true,
			},
		},
		{
			name:              "review フェーズが正しく定義されている",
			phaseName:         "review",
			expectedTrigger:   domain.LabelReviewRequested,
			expectedExecution: domain.LabelReviewing,
			expectedType:      domain.ExecutionTypeCommand,
			expectedPane:      true,
			expectedWorktree:  false,
			expectedCompletions: map[string]bool{
				domain.LabelDone:            true,
				domain.LabelRequiresChanges: true,
			},
		},
		{
			name:              "revise フェーズが正しく定義されている",
			phaseName:         "revise",
			expectedTrigger:   domain.LabelRequiresChanges,
			expectedExecution: domain.LabelRevising,
			expectedType:      domain.ExecutionTypeCommand,
			expectedPane:      true,
			expectedWorktree:  true,
			expectedCompletions: map[string]bool{
				domain.LabelReviewRequested: true,
			},
		},
		{
			name:              "merge フェーズが正しく定義されている",
			phaseName:         "merge",
			expectedTrigger:   domain.LabelDone,
			expectedExecution: domain.LabelMerged,
			expectedType:      domain.ExecutionTypeLabelOnly,
			expectedPane:      false,
			expectedWorktree:  false,
			expectedCompletions: map[string]bool{
				domain.LabelMerged: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := domain.PhaseDefinitions[tt.phaseName]
			require.NotNil(t, phase, "フェーズ %s が定義されていない", tt.phaseName)

			assert.Equal(t, tt.phaseName, phase.Name)
			assert.Equal(t, tt.expectedTrigger, phase.TriggerLabel)
			assert.Equal(t, tt.expectedExecution, phase.ExecutionLabel)
			assert.Equal(t, tt.expectedType, phase.ExecutionType)
			assert.Equal(t, tt.expectedPane, phase.RequiresPane)
			assert.Equal(t, tt.expectedWorktree, phase.RequiresWorktree)

			// 完了ラベルの確認
			for label, shouldExist := range tt.expectedCompletions {
				_, exists := phase.CompletionLabels[label]
				assert.Equal(t, shouldExist, exists, "完了ラベル %s の存在状態が期待と異なる", label)
			}
		})
	}
}

func TestGetPhaseByTrigger(t *testing.T) {
	tests := []struct {
		name          string
		label         string
		expectedPhase string
		expectNil     bool
	}{
		{
			name:          "soba:todoでqueueフェーズを取得",
			label:         domain.LabelTodo,
			expectedPhase: "queue",
		},
		{
			name:          "soba:queuedでplanフェーズを取得",
			label:         domain.LabelQueued,
			expectedPhase: "plan",
		},
		{
			name:          "soba:readyでimplementフェーズを取得",
			label:         domain.LabelReady,
			expectedPhase: "implement",
		},
		{
			name:          "soba:review-requestedでreviewフェーズを取得",
			label:         domain.LabelReviewRequested,
			expectedPhase: "review",
		},
		{
			name:          "soba:requires-changesでreviseフェーズを取得",
			label:         domain.LabelRequiresChanges,
			expectedPhase: "revise",
		},
		{
			name:          "soba:doneでmergeフェーズを取得",
			label:         domain.LabelDone,
			expectedPhase: "merge",
		},
		{
			name:      "存在しないラベルではnilを返す",
			label:     "invalid-label",
			expectNil: true,
		},
		{
			name:      "実行中ラベルではnilを返す",
			label:     domain.LabelPlanning,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := domain.GetPhaseByTrigger(tt.label)

			if tt.expectNil {
				assert.Nil(t, phase)
			} else {
				require.NotNil(t, phase)
				assert.Equal(t, tt.expectedPhase, phase.Name)
			}
		})
	}
}

func TestGetPhaseByExecutionLabel(t *testing.T) {
	tests := []struct {
		name          string
		label         string
		expectedPhase string
		expectNil     bool
	}{
		{
			name:          "soba:queuedでqueueフェーズを取得",
			label:         domain.LabelQueued,
			expectedPhase: "queue",
		},
		{
			name:          "soba:planningでplanフェーズを取得",
			label:         domain.LabelPlanning,
			expectedPhase: "plan",
		},
		{
			name:          "soba:doingでimplementフェーズを取得",
			label:         domain.LabelDoing,
			expectedPhase: "implement",
		},
		{
			name:          "soba:reviewingでreviewフェーズを取得",
			label:         domain.LabelReviewing,
			expectedPhase: "review",
		},
		{
			name:          "soba:revisingでreviseフェーズを取得",
			label:         domain.LabelRevising,
			expectedPhase: "revise",
		},
		{
			name:          "soba:mergedでmergeフェーズを取得",
			label:         domain.LabelMerged,
			expectedPhase: "merge",
		},
		{
			name:      "存在しないラベルではnilを返す",
			label:     "invalid-label",
			expectNil: true,
		},
		{
			name:      "トリガーラベルではnilを返す",
			label:     domain.LabelTodo,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := domain.GetPhaseByExecutionLabel(tt.label)

			if tt.expectNil {
				assert.Nil(t, phase)
			} else {
				require.NotNil(t, phase)
				assert.Equal(t, tt.expectedPhase, phase.Name)
			}
		})
	}
}

func TestIsCompletionLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		expected bool
	}{
		{
			name:     "soba:readyは完了ラベル",
			label:    domain.LabelReady,
			expected: true,
		},
		{
			name:     "soba:review-requestedは完了ラベル",
			label:    domain.LabelReviewRequested,
			expected: true,
		},
		{
			name:     "soba:doneは完了ラベル",
			label:    domain.LabelDone,
			expected: true,
		},
		{
			name:     "soba:requires-changesは完了ラベル",
			label:    domain.LabelRequiresChanges,
			expected: true,
		},
		{
			name:     "soba:mergedは完了ラベル",
			label:    domain.LabelMerged,
			expected: true,
		},
		{
			name:     "soba:queuedは完了ラベル（queueフェーズの）",
			label:    domain.LabelQueued,
			expected: true,
		},
		{
			name:     "soba:todoはトリガーラベルなので完了ラベルではない",
			label:    domain.LabelTodo,
			expected: false,
		},
		{
			name:     "soba:planningは実行中ラベルなので完了ラベルではない",
			label:    domain.LabelPlanning,
			expected: false,
		},
		{
			name:     "soba:doingは実行中ラベルなので完了ラベルではない",
			label:    domain.LabelDoing,
			expected: false,
		},
		{
			name:     "存在しないラベルは完了ラベルではない",
			label:    "invalid-label",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.IsCompletionLabel(tt.label)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNextActionForCompletion(t *testing.T) {
	tests := []struct {
		name              string
		executionLabel    string
		completionLabel   string
		expectedRemove    string
		expectedAuto      bool
		expectedNextPhase string
		expectNil         bool
	}{
		{
			name:              "planフェーズ完了（ready）",
			executionLabel:    domain.LabelPlanning,
			completionLabel:   domain.LabelReady,
			expectedRemove:    domain.LabelPlanning,
			expectedAuto:      false,
			expectedNextPhase: "",
		},
		{
			name:              "implementフェーズ完了（review-requested）",
			executionLabel:    domain.LabelDoing,
			completionLabel:   domain.LabelReviewRequested,
			expectedRemove:    domain.LabelDoing,
			expectedAuto:      false,
			expectedNextPhase: "",
		},
		{
			name:              "reviewフェーズ完了（done）",
			executionLabel:    domain.LabelReviewing,
			completionLabel:   domain.LabelDone,
			expectedRemove:    domain.LabelReviewing,
			expectedAuto:      false,
			expectedNextPhase: "",
		},
		{
			name:              "reviewフェーズ完了（requires-changes）",
			executionLabel:    domain.LabelReviewing,
			completionLabel:   domain.LabelRequiresChanges,
			expectedRemove:    domain.LabelReviewing,
			expectedAuto:      false,
			expectedNextPhase: "",
		},
		{
			name:              "queueフェーズの自動遷移",
			executionLabel:    domain.LabelQueued,
			completionLabel:   domain.LabelQueued,
			expectedRemove:    "",
			expectedAuto:      true,
			expectedNextPhase: "plan",
		},
		{
			name:            "存在しない実行ラベル",
			executionLabel:  "invalid-label",
			completionLabel: domain.LabelReady,
			expectNil:       true,
		},
		{
			name:            "不正な完了ラベル",
			executionLabel:  domain.LabelPlanning,
			completionLabel: domain.LabelDone, // planフェーズにはdoneはない
			expectNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := domain.GetNextActionForCompletion(tt.executionLabel, tt.completionLabel)

			if tt.expectNil {
				assert.Nil(t, action)
			} else {
				require.NotNil(t, action)
				assert.Equal(t, tt.expectedRemove, action.RemoveLabel)
				assert.Equal(t, tt.expectedAuto, action.AutoTransition)
				assert.Equal(t, tt.expectedNextPhase, action.NextPhase)
			}
		})
	}
}

func TestPhaseAutoTransition(t *testing.T) {
	tests := []struct {
		name         string
		phaseName    string
		expectedAuto bool
		expectedNext string
	}{
		{
			name:         "queueフェーズは自動でplanへ遷移",
			phaseName:    "queue",
			expectedAuto: true,
			expectedNext: "plan",
		},
		{
			name:         "planフェーズは自動遷移しない",
			phaseName:    "plan",
			expectedAuto: false,
		},
		{
			name:         "implementフェーズは自動遷移しない",
			phaseName:    "implement",
			expectedAuto: false,
		},
		{
			name:         "reviewフェーズは自動遷移しない",
			phaseName:    "review",
			expectedAuto: false,
		},
		{
			name:         "reviseフェーズは自動遷移しない",
			phaseName:    "revise",
			expectedAuto: false,
		},
		{
			name:         "mergeフェーズは自動遷移しない",
			phaseName:    "merge",
			expectedAuto: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := domain.PhaseDefinitions[tt.phaseName]
			require.NotNil(t, phase)

			// 最初の完了アクションを確認
			for _, action := range phase.CompletionLabels {
				assert.Equal(t, tt.expectedAuto, action.AutoTransition)
				if tt.expectedAuto {
					assert.Equal(t, tt.expectedNext, action.NextPhase)
				}
				break // 最初のアクションのみチェック
			}
		})
	}
}
