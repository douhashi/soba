package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/domain"
)

func TestPhaseConstants(t *testing.T) {
	tests := []struct {
		name  string
		phase domain.Phase
		want  string
	}{
		{
			name:  "PhaseQueue定数が正しい値を持つ",
			phase: domain.PhaseQueue,
			want:  "queue",
		},
		{
			name:  "PhasePlan定数が正しい値を持つ",
			phase: domain.PhasePlan,
			want:  "plan",
		},
		{
			name:  "PhaseImplement定数が正しい値を持つ",
			phase: domain.PhaseImplement,
			want:  "implement",
		},
		{
			name:  "PhaseReview定数が正しい値を持つ",
			phase: domain.PhaseReview,
			want:  "review",
		},
		{
			name:  "PhaseRevise定数が正しい値を持つ",
			phase: domain.PhaseRevise,
			want:  "revise",
		},
		{
			name:  "PhaseMerge定数が正しい値を持つ",
			phase: domain.PhaseMerge,
			want:  "merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.phase))
		})
	}
}

func TestLabelConstants(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{
			name:  "LabelTodo定数が正しい値を持つ",
			label: domain.LabelTodo,
			want:  "soba:todo",
		},
		{
			name:  "LabelQueued定数が正しい値を持つ",
			label: domain.LabelQueued,
			want:  "soba:queued",
		},
		{
			name:  "LabelPlanning定数が正しい値を持つ",
			label: domain.LabelPlanning,
			want:  "soba:planning",
		},
		{
			name:  "LabelReady定数が正しい値を持つ",
			label: domain.LabelReady,
			want:  "soba:ready",
		},
		{
			name:  "LabelDoing定数が正しい値を持つ",
			label: domain.LabelDoing,
			want:  "soba:doing",
		},
		{
			name:  "LabelReviewRequested定数が正しい値を持つ",
			label: domain.LabelReviewRequested,
			want:  "soba:review-requested",
		},
		{
			name:  "LabelReviewing定数が正しい値を持つ",
			label: domain.LabelReviewing,
			want:  "soba:reviewing",
		},
		{
			name:  "LabelDone定数が正しい値を持つ",
			label: domain.LabelDone,
			want:  "soba:done",
		},
		{
			name:  "LabelRequiresChanges定数が正しい値を持つ",
			label: domain.LabelRequiresChanges,
			want:  "soba:requires-changes",
		},
		{
			name:  "LabelRevising定数が正しい値を持つ",
			label: domain.LabelRevising,
			want:  "soba:revising",
		},
		{
			name:  "LabelMerged定数が正しい値を持つ",
			label: domain.LabelMerged,
			want:  "soba:merged",
		},
		{
			name:  "LabelLGTM定数が正しい値を持つ",
			label: domain.LabelLGTM,
			want:  "soba:lgtm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.label)
		})
	}
}

func TestPhaseTransition(t *testing.T) {
	tests := []struct {
		name       string
		phase      domain.Phase
		wantFrom   string
		wantTo     string
		wantAction string
	}{
		{
			name:       "PhaseQueueの遷移情報が正しい",
			phase:      domain.PhaseQueue,
			wantFrom:   domain.LabelTodo,
			wantTo:     domain.LabelQueued,
			wantAction: "queue",
		},
		{
			name:       "PhasePlanの遷移情報が正しい",
			phase:      domain.PhasePlan,
			wantFrom:   domain.LabelQueued,
			wantTo:     domain.LabelReady,
			wantAction: "plan",
		},
		{
			name:       "PhaseImplementの遷移情報が正しい",
			phase:      domain.PhaseImplement,
			wantFrom:   domain.LabelReady,
			wantTo:     domain.LabelReviewRequested,
			wantAction: "implement",
		},
		{
			name:       "PhaseReviewの遷移情報が正しい",
			phase:      domain.PhaseReview,
			wantFrom:   domain.LabelReviewRequested,
			wantTo:     domain.LabelDone,
			wantAction: "review",
		},
		{
			name:       "PhaseReviseの遷移情報が正しい",
			phase:      domain.PhaseRevise,
			wantFrom:   domain.LabelRequiresChanges,
			wantTo:     domain.LabelReviewRequested,
			wantAction: "revise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transition := domain.GetTransition(tt.phase)
			require.NotNil(t, transition)
			assert.Equal(t, tt.wantFrom, transition.From)
			assert.Equal(t, tt.wantTo, transition.To)
			assert.Equal(t, tt.wantAction, transition.Action)
		})
	}
}

func TestPhaseStrategy_GetCurrentPhase(t *testing.T) {
	strategy := domain.NewDefaultPhaseStrategy()

	tests := []struct {
		name    string
		labels  []string
		want    domain.Phase
		wantErr bool
	}{
		{
			name:   "soba:todoラベルからPhaseQueueを判定",
			labels: []string{"bug", domain.LabelTodo, "priority:high"},
			want:   domain.PhaseQueue,
		},
		{
			name:   "soba:queuedラベルからPhasePlanを判定",
			labels: []string{domain.LabelQueued},
			want:   domain.PhasePlan,
		},
		{
			name:   "soba:planningラベルからPhasePlanを判定",
			labels: []string{domain.LabelPlanning},
			want:   domain.PhasePlan,
		},
		{
			name:   "soba:readyラベルからPhaseImplementを判定",
			labels: []string{domain.LabelReady},
			want:   domain.PhaseImplement,
		},
		{
			name:   "soba:doingラベルからPhaseImplementを判定",
			labels: []string{domain.LabelDoing},
			want:   domain.PhaseImplement,
		},
		{
			name:   "soba:review-requestedラベルからPhaseReviewを判定",
			labels: []string{domain.LabelReviewRequested},
			want:   domain.PhaseReview,
		},
		{
			name:   "soba:reviewingラベルからPhaseReviewを判定",
			labels: []string{domain.LabelReviewing},
			want:   domain.PhaseReview,
		},
		{
			name:   "soba:requires-changesラベルからPhaseReviseを判定",
			labels: []string{domain.LabelRequiresChanges},
			want:   domain.PhaseRevise,
		},
		{
			name:   "soba:revisingラベルからPhaseReviseを判定",
			labels: []string{domain.LabelRevising},
			want:   domain.PhaseRevise,
		},
		{
			name:   "soba:doneラベルからPhaseMergeを判定",
			labels: []string{domain.LabelDone},
			want:   domain.PhaseMerge,
		},
		{
			name:   "soba:mergedラベルからPhaseMergeを判定",
			labels: []string{domain.LabelMerged},
			want:   domain.PhaseMerge,
		},
		{
			name:    "複数のsobaラベルがある場合はエラー",
			labels:  []string{domain.LabelTodo, domain.LabelDoing},
			wantErr: true,
		},
		{
			name:    "sobaラベルがない場合はエラー",
			labels:  []string{"bug", "priority:high"},
			wantErr: true,
		},
		{
			name:   "soba:lgtmは無視される",
			labels: []string{domain.LabelDoing, domain.LabelLGTM},
			want:   domain.PhaseImplement,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.GetCurrentPhase(tt.labels)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPhaseStrategy_GetNextLabel(t *testing.T) {
	strategy := domain.NewDefaultPhaseStrategy()

	tests := []struct {
		name    string
		phase   domain.Phase
		want    string
		wantErr bool
	}{
		{
			name:  "PhaseQueueの次はsoba:queued",
			phase: domain.PhaseQueue,
			want:  domain.LabelQueued,
		},
		{
			name:  "PhasePlanの次はsoba:ready",
			phase: domain.PhasePlan,
			want:  domain.LabelReady,
		},
		{
			name:  "PhaseImplementの次はsoba:review-requested",
			phase: domain.PhaseImplement,
			want:  domain.LabelReviewRequested,
		},
		{
			name:  "PhaseReviewの次はsoba:done",
			phase: domain.PhaseReview,
			want:  domain.LabelDone,
		},
		{
			name:  "PhaseReviseの次はsoba:review-requested",
			phase: domain.PhaseRevise,
			want:  domain.LabelReviewRequested,
		},
		{
			name:    "PhaseMergeには次の遷移がない",
			phase:   domain.PhaseMerge,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.GetNextLabel(tt.phase)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPhaseStrategy_ValidateTransition(t *testing.T) {
	strategy := domain.NewDefaultPhaseStrategy()

	tests := []struct {
		name    string
		from    domain.Phase
		to      domain.Phase
		wantErr bool
	}{
		{
			name: "PhaseQueueからPhasePlanへの遷移は有効",
			from: domain.PhaseQueue,
			to:   domain.PhasePlan,
		},
		{
			name: "PhasePlanからPhaseImplementへの遷移は有効",
			from: domain.PhasePlan,
			to:   domain.PhaseImplement,
		},
		{
			name: "PhaseImplementからPhaseReviewへの遷移は有効",
			from: domain.PhaseImplement,
			to:   domain.PhaseReview,
		},
		{
			name: "PhaseReviewからPhaseMergeへの遷移は有効",
			from: domain.PhaseReview,
			to:   domain.PhaseMerge,
		},
		{
			name: "PhaseReviewからPhaseReviseへの遷移は有効",
			from: domain.PhaseReview,
			to:   domain.PhaseRevise,
		},
		{
			name: "PhaseReviseからPhaseReviewへの遷移は有効",
			from: domain.PhaseRevise,
			to:   domain.PhaseReview,
		},
		{
			name:    "PhaseQueueからPhaseReviewへの直接遷移は無効",
			from:    domain.PhaseQueue,
			to:      domain.PhaseReview,
			wantErr: true,
		},
		{
			name:    "PhaseMergeからの遷移は無効",
			from:    domain.PhaseMerge,
			to:      domain.PhaseQueue,
			wantErr: true,
		},
		{
			name:    "PhaseImplementからPhaseQueueへの逆方向遷移は無効",
			from:    domain.PhaseImplement,
			to:      domain.PhaseQueue,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := strategy.ValidateTransition(tt.from, tt.to)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
