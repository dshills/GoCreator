package providers

import (
	"context"
	"fmt"
	"sync"
)

// Validator validates provider credentials in parallel
type Validator struct {
	providers map[string]LLMProvider
}

// NewValidator creates a new validator for the given providers
func NewValidator(providers map[string]LLMProvider) *Validator {
	return &Validator{
		providers: providers,
	}
}

// ValidateAll validates all providers in parallel with timeout and error aggregation
func (v *Validator) ValidateAll(ctx context.Context) error {
	if len(v.providers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(v.providers))

	// Validate each provider in parallel
	for id, provider := range v.providers {
		wg.Add(1)
		go func(id string, p LLMProvider) {
			defer wg.Done()

			// Initialize provider (validates credentials)
			if err := p.Initialize(ctx); err != nil {
				errCh <- fmt.Errorf("provider %s: %w", id, err)
			}
		}(id, provider)
	}

	// Wait for all validations to complete
	wg.Wait()
	close(errCh)

	// Collect all errors
	errors := make([]error, 0, len(v.providers))
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("credential validation failed for %d provider(s): %v", len(errors), errors)
	}

	return nil
}
