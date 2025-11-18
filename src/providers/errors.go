package providers

import "fmt"

// ProviderError represents an error from an LLM provider operation
type ProviderError struct {
	ProviderID string    // Provider that generated the error
	Code       ErrorCode // Error classification code
	Message    string    // Human-readable error message
	Cause      error     // Underlying error
	Retryable  bool      // Whether the error is retryable
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.ProviderID, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.ProviderID, e.Code, e.Message)
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// NewProviderError creates a new ProviderError with automatic retryability classification
func NewProviderError(providerID string, code ErrorCode, message string, cause error) *ProviderError {
	return &ProviderError{
		ProviderID: providerID,
		Code:       code,
		Message:    message,
		Cause:      cause,
		Retryable:  isRetryableErrorCode(code),
	}
}

// isRetryableErrorCode determines if an error code represents a retryable error
func isRetryableErrorCode(code ErrorCode) bool {
	switch code {
	case ErrorCodeRateLimit, ErrorCodeNetwork, ErrorCodeTimeout, ErrorCodeServerError:
		return true
	case ErrorCodeAuth, ErrorCodeInvalidInput, ErrorCodeUnknown:
		return false
	default:
		return false
	}
}

// ConfigError represents an error in configuration validation or loading
type ConfigError struct {
	Field   string // Configuration field with error
	Message string // Error description
	Cause   error  // Underlying error
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	if e.Field != "" {
		if e.Cause != nil {
			return fmt.Sprintf("config error in field '%s': %s (caused by: %v)", e.Field, e.Message, e.Cause)
		}
		return fmt.Sprintf("config error in field '%s': %s", e.Field, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("config error: %s (caused by: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new ConfigError
func NewConfigError(field, message string, cause error) *ConfigError {
	return &ConfigError{
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}
