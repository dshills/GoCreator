// Package models defines the core data structures used throughout GoCreator.
package models

import (
	"fmt"
	"time"
)

// Option represents a clarification option for a question
type Option struct {
	Label        string `json:"label"`
	Description  string `json:"description,omitempty"`
	Implications string `json:"implications,omitempty"`
}

// Question represents a clarification question
type Question struct {
	ID         string   `json:"id"`
	Topic      string   `json:"topic,omitempty"`
	Context    string   `json:"context,omitempty"`
	Question   string   `json:"question"`
	Options    []Option `json:"options"`
	UserAnswer *string  `json:"user_answer,omitempty"`
}

// Ambiguity represents an identified ambiguity in the specification
type Ambiguity struct {
	Type        string `json:"type"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description"`
	Severity    string `json:"severity,omitempty"`
}

// ClarificationRequest represents a request for clarification
type ClarificationRequest struct {
	SchemaVersion string      `json:"schema_version"`
	ID            string      `json:"id"`
	SpecID        string      `json:"spec_id"`
	Questions     []Question  `json:"questions"`
	Ambiguities   []Ambiguity `json:"ambiguities"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Validate validates the clarification request
func (c *ClarificationRequest) Validate() error {
	if len(c.Questions) == 0 && len(c.Ambiguities) == 0 {
		return fmt.Errorf("clarification request must have at least 1 question or ambiguity")
	}

	for i, q := range c.Questions {
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return fmt.Errorf("question %d must have 2-4 options, got %d", i, len(q.Options))
		}
	}

	return nil
}

// Answer represents an answer to a clarification question
type Answer struct {
	QuestionID     string  `json:"question_id"`
	SelectedOption *string `json:"selected_option,omitempty"`
	CustomAnswer   *string `json:"custom_answer,omitempty"`
}

// ClarificationResponse represents a user's response to clarification questions
type ClarificationResponse struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	RequestID     string            `json:"request_id"`
	Answers       map[string]Answer `json:"answers"`
	AnsweredAt    time.Time         `json:"answered_at"`
}

// ValidateAgainst validates the response against a clarification request
func (r *ClarificationResponse) ValidateAgainst(request *ClarificationRequest) error {
	// Check that all questions are answered
	for _, q := range request.Questions {
		answer, ok := r.Answers[q.ID]
		if !ok {
			return fmt.Errorf("must answer all questions: missing answer for %s", q.ID)
		}

		// Check that answer has either selected option OR custom answer, not both
		if answer.SelectedOption != nil && answer.CustomAnswer != nil {
			return fmt.Errorf("answer for %s must have either SelectedOption OR CustomAnswer, not both", q.ID)
		}

		if answer.SelectedOption == nil && answer.CustomAnswer == nil {
			return fmt.Errorf("answer for %s must have either SelectedOption or CustomAnswer", q.ID)
		}
	}

	return nil
}
