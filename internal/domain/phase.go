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
	PhaseMerge     Phase = "merge"
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
	LabelMerged          = "soba:merged"
	LabelLGTM            = "soba:lgtm"
)

// PhaseTransition represents a workflow transition
type PhaseTransition struct {
	From   string
	To     string
	Action string
}

// transitions defines the workflow transitions for each phase
var transitions = map[Phase]PhaseTransition{
	PhaseQueue: {
		From:   LabelTodo,
		To:     LabelQueued,
		Action: "queue",
	},
	PhasePlan: {
		From:   LabelQueued,
		To:     LabelReady,
		Action: "plan",
	},
	PhaseImplement: {
		From:   LabelReady,
		To:     LabelReviewRequested,
		Action: "implement",
	},
	PhaseReview: {
		From:   LabelReviewRequested,
		To:     LabelDone,
		Action: "review",
	},
	PhaseRevise: {
		From:   LabelRequiresChanges,
		To:     LabelReviewRequested,
		Action: "revise",
	},
}

// GetTransition returns the transition for the given phase
func GetTransition(phase Phase) *PhaseTransition {
	if transition, ok := transitions[phase]; ok {
		return &transition
	}
	return nil
}

// PhaseStrategy defines the interface for phase management
type PhaseStrategy interface {
	// GetCurrentPhase determines the current phase from labels
	GetCurrentPhase(labels []string) (Phase, error)
	// GetNextLabel returns the next label for the given phase
	GetNextLabel(currentPhase Phase) (string, error)
	// ValidateTransition validates if a transition from one phase to another is valid
	ValidateTransition(from, to Phase) error
}

// DefaultPhaseStrategy is the default implementation of PhaseStrategy
type DefaultPhaseStrategy struct {
	labelToPhase map[string]Phase
}

// NewDefaultPhaseStrategy creates a new DefaultPhaseStrategy
func NewDefaultPhaseStrategy() PhaseStrategy {
	return &DefaultPhaseStrategy{
		labelToPhase: map[string]Phase{
			LabelTodo:            PhaseQueue,
			LabelQueued:          PhasePlan,
			LabelPlanning:        PhasePlan,
			LabelReady:           PhaseImplement,
			LabelDoing:           PhaseImplement,
			LabelReviewRequested: PhaseReview,
			LabelReviewing:       PhaseReview,
			LabelRequiresChanges: PhaseRevise,
			LabelRevising:        PhaseRevise,
			LabelDone:            PhaseMerge,
			LabelMerged:          PhaseMerge,
		},
	}
}

// GetCurrentPhase determines the current phase from labels
func (s *DefaultPhaseStrategy) GetCurrentPhase(labels []string) (Phase, error) {
	var sobaLabels []string
	for _, label := range labels {
		if strings.HasPrefix(label, "soba:") && label != LabelLGTM {
			sobaLabels = append(sobaLabels, label)
		}
	}

	if len(sobaLabels) == 0 {
		return "", fmt.Errorf("no soba label found")
	}

	if len(sobaLabels) > 1 {
		return "", fmt.Errorf("multiple soba labels found: %v", sobaLabels)
	}

	phase, ok := s.labelToPhase[sobaLabels[0]]
	if !ok {
		return "", fmt.Errorf("unknown soba label: %s", sobaLabels[0])
	}

	return phase, nil
}

// GetNextLabel returns the next label for the given phase
func (s *DefaultPhaseStrategy) GetNextLabel(currentPhase Phase) (string, error) {
	transition := GetTransition(currentPhase)
	if transition == nil {
		return "", fmt.Errorf("no transition defined for phase: %s", currentPhase)
	}
	return transition.To, nil
}

// ValidateTransition validates if a transition from one phase to another is valid
func (s *DefaultPhaseStrategy) ValidateTransition(from, to Phase) error {
	// Define valid transitions
	validTransitions := map[Phase][]Phase{
		PhaseQueue:     {PhasePlan},
		PhasePlan:      {PhaseImplement},
		PhaseImplement: {PhaseReview},
		PhaseReview:    {PhaseMerge, PhaseRevise},
		PhaseRevise:    {PhaseReview},
		PhaseMerge:     {}, // No transitions from merge
	}

	validToPhases, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("invalid from phase: %s", from)
	}

	for _, validTo := range validToPhases {
		if validTo == to {
			return nil
		}
	}

	if len(validToPhases) == 0 {
		return fmt.Errorf("no valid transitions from phase: %s", from)
	}

	return fmt.Errorf("invalid transition from %s to %s", from, to)
}
