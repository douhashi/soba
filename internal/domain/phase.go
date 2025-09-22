package domain

import (
	"fmt"
	"strings"
)

// Phase represents the current phase in the workflow
type Phase string

const (
	PhaseQueue     Phase = "queue"
	PhasePlan      Phase = "plan"
	PhaseImplement Phase = "implement"
	PhaseReview    Phase = "review"
	PhaseRevise    Phase = "revise"
)

// PhaseExecutionType represents how a phase should be executed
type PhaseExecutionType string

const (
	// ExecutionTypeLabelOnly only updates labels and transitions immediately
	ExecutionTypeLabelOnly PhaseExecutionType = "label_only"
	// ExecutionTypeCommand updates labels and executes commands
	ExecutionTypeCommand PhaseExecutionType = "command"
)

// Label constants for soba workflow
const (
	LabelTodo            = "soba:todo"
	LabelQueued          = "soba:queued"
	LabelPlanning        = "soba:planning"
	LabelReady           = "soba:ready"
	LabelDoing           = "soba:doing"
	LabelReviewRequested = "soba:review-requested"
	LabelReviewing       = "soba:reviewing"
	LabelDone            = "soba:done"
	LabelRequiresChanges = "soba:requires-changes"
	LabelRevising        = "soba:revising"
	LabelLGTM            = "soba:lgtm"
)

// PhaseDefinition はフェーズの完全な定義を表す
type PhaseDefinition struct {
	Name             string                // フェーズ名
	TriggerLabel     string                // フェーズ開始のトリガーラベル
	ExecutionLabel   string                // sobaが設定する実行中ラベル
	ExecutionType    PhaseExecutionType    // 実行タイプ（ラベルのみ or コマンド実行）
	RequiresPane     bool                  // tmuxペインが必要か
	RequiresWorktree bool                  // gitワークツリーが必要か
	CompletionLabels map[string]NextAction // 完了ラベルと次のアクション
}

// NextAction は完了ラベルに対応する次のアクションを定義
type NextAction struct {
	RemoveLabel    string // 削除するラベル（実行中ラベルなど）
	AutoTransition bool   // 自動的に次フェーズに遷移するか
	NextPhase      string // 次のフェーズ名（自動遷移時）
}

// PhaseDefinitions は全フェーズの定義
var PhaseDefinitions = map[string]*PhaseDefinition{
	"queue": {
		Name:             "queue",
		TriggerLabel:     LabelTodo,
		ExecutionLabel:   LabelQueued,
		ExecutionType:    ExecutionTypeLabelOnly,
		RequiresPane:     false,
		RequiresWorktree: false,
		CompletionLabels: map[string]NextAction{
			LabelQueued: { // queuedになったら即座にplanへ
				RemoveLabel:    "",
				AutoTransition: true,
				NextPhase:      "plan",
			},
		},
	},
	"plan": {
		Name:             "plan",
		TriggerLabel:     LabelQueued,
		ExecutionLabel:   LabelPlanning,
		ExecutionType:    ExecutionTypeCommand,
		RequiresPane:     true,
		RequiresWorktree: true,
		CompletionLabels: map[string]NextAction{
			LabelReady: { // 外部ツールがreadyを設定
				RemoveLabel:    LabelPlanning,
				AutoTransition: false, // ready状態で人間の判断を待つ
				NextPhase:      "",
			},
		},
	},
	"implement": {
		Name:             "implement",
		TriggerLabel:     LabelReady,
		ExecutionLabel:   LabelDoing,
		ExecutionType:    ExecutionTypeCommand,
		RequiresPane:     true,
		RequiresWorktree: true,
		CompletionLabels: map[string]NextAction{
			LabelReviewRequested: { // 外部ツールがPR作成後に設定
				RemoveLabel:    LabelDoing,
				AutoTransition: false,
				NextPhase:      "",
			},
		},
	},
	"review": {
		Name:             "review",
		TriggerLabel:     LabelReviewRequested,
		ExecutionLabel:   LabelReviewing,
		ExecutionType:    ExecutionTypeCommand,
		RequiresPane:     true,
		RequiresWorktree: false,
		CompletionLabels: map[string]NextAction{
			LabelDone: { // レビュー承認
				RemoveLabel:    LabelReviewing,
				AutoTransition: false,
				NextPhase:      "",
			},
			LabelRequiresChanges: { // 修正要求
				RemoveLabel:    LabelReviewing,
				AutoTransition: false,
				NextPhase:      "",
			},
		},
	},
	"revise": {
		Name:             "revise",
		TriggerLabel:     LabelRequiresChanges,
		ExecutionLabel:   LabelRevising,
		ExecutionType:    ExecutionTypeCommand,
		RequiresPane:     true,
		RequiresWorktree: true,
		CompletionLabels: map[string]NextAction{
			LabelReviewRequested: { // 修正後に再レビュー
				RemoveLabel:    LabelRevising,
				AutoTransition: false,
				NextPhase:      "",
			},
		},
	},
}

// GetPhaseByTrigger はトリガーラベルから対応するフェーズ定義を取得
func GetPhaseByTrigger(label string) *PhaseDefinition {
	for _, phase := range PhaseDefinitions {
		if phase.TriggerLabel == label {
			return phase
		}
	}
	return nil
}

// GetPhaseByExecutionLabel は実行中ラベルから対応するフェーズ定義を取得
func GetPhaseByExecutionLabel(label string) *PhaseDefinition {
	for _, phase := range PhaseDefinitions {
		if phase.ExecutionLabel == label {
			return phase
		}
	}
	return nil
}

// IsCompletionLabel は指定されたラベルが何らかのフェーズの完了ラベルかチェック
func IsCompletionLabel(label string) bool {
	for _, phase := range PhaseDefinitions {
		if _, ok := phase.CompletionLabels[label]; ok {
			return true
		}
	}
	return false
}

// GetNextActionForCompletion は完了ラベルに対応する次のアクションを取得
func GetNextActionForCompletion(executionLabel, completionLabel string) *NextAction {
	phase := GetPhaseByExecutionLabel(executionLabel)
	if phase == nil {
		return nil
	}

	if action, ok := phase.CompletionLabels[completionLabel]; ok {
		return &action
	}
	return nil
}

// GetCurrentPhaseFromLabels はラベルリストから現在のフェーズを判定する
func GetCurrentPhaseFromLabels(labels []string) (Phase, error) {
	// soba:で始まるラベルを探す（LGTMは除く）
	var sobaLabel string
	for _, label := range labels {
		if strings.HasPrefix(label, "soba:") && label != LabelLGTM {
			if sobaLabel != "" {
				// 複数のsobaラベルがある場合はエラー
				return "", fmt.Errorf("multiple soba labels found")
			}
			sobaLabel = label
		}
	}

	if sobaLabel == "" {
		return "", fmt.Errorf("no soba label found")
	}

	// トリガーラベルから判定
	if phase := GetPhaseByTrigger(sobaLabel); phase != nil {
		return Phase(phase.Name), nil
	}

	// 実行中ラベルから判定
	if phase := GetPhaseByExecutionLabel(sobaLabel); phase != nil {
		return Phase(phase.Name), nil
	}

	// 完了ラベルから判定（どのフェーズの完了ラベルか特定）
	// review-requestedやrequires-changesなど、複数のフェーズで使われるラベルがある
	switch sobaLabel {
	case LabelReady:
		return PhaseImplement, nil // readyはimplementフェーズのトリガー
	case LabelReviewRequested:
		// review-requestedはimplementまたはreviseの完了後
		// この場合、Reviewフェーズとみなす
		return PhaseReview, nil
	case LabelDone:
		return PhaseReview, nil // doneはreviewの完了後
	case LabelRequiresChanges:
		return PhaseRevise, nil // requires-changesはreviewの完了後
	}

	return "", fmt.Errorf("unknown soba label: %s", sobaLabel)
}
