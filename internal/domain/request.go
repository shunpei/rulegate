package domain

import "fmt"

// AskRequest is the JSON body for POST /ask.
type AskRequest struct {
	QuestionJA  string         `json:"question_ja"`
	Discipline  string         `json:"discipline"`
	RuleEdition string         `json:"rule_edition"`
	Context     *QueryContext  `json:"context,omitempty"`
	Options     *RequestOption `json:"options,omitempty"`
}

type QueryContext struct {
	BoatClass string `json:"boat_class,omitempty"`
	RacePhase string `json:"race_phase,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

type RequestOption struct {
	TopK           *int    `json:"top_k,omitempty"`
	MinConfidence  *float64 `json:"min_confidence,omitempty"`
	ReturnContexts bool    `json:"return_contexts,omitempty"`
	AnswerStyle    string  `json:"answer_style,omitempty"`
}

const (
	DefaultDiscipline  = "canoe_slalom"
	DefaultRuleEdition = "2025"
	MaxQuestionLen     = 1000
)

// Validate checks required fields and applies defaults.
func (r *AskRequest) Validate() error {
	if r.QuestionJA == "" {
		return NewValidationError("question_ja is required")
	}
	if len([]rune(r.QuestionJA)) > MaxQuestionLen {
		return NewValidationError(fmt.Sprintf("question_ja must be <= %d characters", MaxQuestionLen))
	}
	if r.Discipline == "" {
		r.Discipline = DefaultDiscipline
	}
	if r.RuleEdition == "" {
		r.RuleEdition = DefaultRuleEdition
	}
	return nil
}

// EffectiveTopK returns the top_k value, falling back to the provided default.
func (r *AskRequest) EffectiveTopK(defaultTopK int) int {
	if r.Options != nil && r.Options.TopK != nil {
		v := *r.Options.TopK
		if v >= 1 && v <= 20 {
			return v
		}
	}
	return defaultTopK
}

// EffectiveMinConfidence returns the min_confidence value, falling back to the provided default.
func (r *AskRequest) EffectiveMinConfidence(defaultMinConf float64) float64 {
	if r.Options != nil && r.Options.MinConfidence != nil {
		v := *r.Options.MinConfidence
		if v >= 0.0 && v <= 1.0 {
			return v
		}
	}
	return defaultMinConf
}
