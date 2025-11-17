package main

import (
	"errors"
	"fmt"
)

// Exit codes as specified in the CLI contract
const (
	ExitCodeSuccess            = 0 // Success
	ExitCodeGeneralError       = 1 // General error (invalid arguments, config errors)
	ExitCodeSpecError          = 2 // Specification parsing/validation error
	ExitCodeClarificationError = 3 // Clarification phase error (LLM provider failure, etc.)
	ExitCodeGenerationError    = 4 // Generation phase error
	ExitCodeValidationError    = 5 // Validation phase error (build/lint/test failures)
	ExitCodeFileSystemError    = 6 // File system error (permission denied, disk full)
	ExitCodeNetworkError       = 7 // Network error (LLM provider unreachable)
	ExitCodeInternalError      = 8 // Internal error (unexpected panic, etc.)
)

// ExitError wraps an error with an exit code
type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

// FormatError formats an error message for display
func FormatError(err error, command string) string {
	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		exitErr = ExitError{Code: ExitCodeGeneralError, Err: err}
	}

	errorType := getErrorType(exitErr.Code)

	msg := fmt.Sprintf("Error: %s\n\n", errorType)
	msg += fmt.Sprintf("%s\n\n", exitErr.Error())
	msg += fmt.Sprintf("For help, run: gocreator %s --help\n", command)

	return msg
}

func getErrorType(code int) string {
	switch code {
	case ExitCodeSpecError:
		return "Specification Validation Failed"
	case ExitCodeClarificationError:
		return "Clarification Phase Failed"
	case ExitCodeGenerationError:
		return "Generation Phase Failed"
	case ExitCodeValidationError:
		return "Validation Phase Failed"
	case ExitCodeFileSystemError:
		return "File System Error"
	case ExitCodeNetworkError:
		return "Network Error"
	case ExitCodeInternalError:
		return "Internal Error"
	default:
		return "Error"
	}
}
