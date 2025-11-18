// Package yamlutil provides enhanced YAML parsing with detailed error messages including
// line numbers, column positions, and suggestions for common issues.
package yamlutil

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseError represents a detailed YAML parsing error with location information
type ParseError struct {
	Line       int    // Line number where error occurred (1-indexed)
	Column     int    // Column number where error occurred (1-indexed)
	Message    string // Error message
	Context    string // Surrounding lines for context
	Suggestion string // Suggestion on how to fix the error
}

// Error implements the error interface
func (e *ParseError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("YAML parse error at line %d, column %d: %s", e.Line, e.Column, e.Message))

	if e.Context != "" {
		sb.WriteString("\n\nContext:\n")
		sb.WriteString(e.Context)
	}

	if e.Suggestion != "" {
		sb.WriteString("\n\nSuggestion: ")
		sb.WriteString(e.Suggestion)
	}

	return sb.String()
}

// Unmarshal parses YAML content with enhanced error reporting
func Unmarshal(data []byte, v interface{}) error {
	err := yaml.Unmarshal(data, v)
	if err == nil {
		return nil
	}

	return enhanceError(err, string(data))
}

// enhanceError converts a yaml.TypeError or other YAML error into a detailed ParseError
func enhanceError(err error, content string) error {
	// Handle yaml.TypeError which contains multiple errors
	if typeErr, ok := err.(*yaml.TypeError); ok {
		// Return the first error with enhancement
		if len(typeErr.Errors) > 0 {
			return parseYAMLError(typeErr.Errors[0], content)
		}
	}

	// Handle other YAML errors
	return parseYAMLError(err.Error(), content)
}

// parseYAMLError extracts line number and creates a detailed ParseError
func parseYAMLError(errMsg string, content string) error {
	// Try to extract line number from error message
	// YAML errors typically have format: "yaml: line X: message" or "line X: message"
	var line, column int
	var message string

	// Try to find "line N:" pattern anywhere in the message
	linePattern := "line "
	if idx := strings.Index(errMsg, linePattern); idx >= 0 {
		// Extract from this point
		remaining := errMsg[idx:]

		// Try to parse line number
		n, err := fmt.Sscanf(remaining, "line %d: %s", &line, &message)
		if n >= 1 && err == nil {
			// Successfully extracted line number
			if message == "" {
				// If we couldn't extract message, take everything after "line N:"
				parts := strings.SplitN(remaining, ": ", 2)
				if len(parts) == 2 {
					message = parts[1]
				} else {
					message = errMsg
				}
			}
		} else {
			// Couldn't parse, use full message
			message = errMsg
			line = 0
		}
	} else {
		// No line number in error, return generic error
		message = errMsg
		line = 0
	}

	// Clean up message
	message = strings.TrimPrefix(message, "yaml: ")
	message = strings.TrimPrefix(message, "line ")
	// Remove "line N:" from beginning if present
	if idx := strings.Index(message, ":"); idx > 0 {
		// Check if everything before : is a number
		before := strings.TrimSpace(message[:idx])
		if _, err := fmt.Sscanf(before, "%d", &line); err == nil {
			// It was a line number, skip it
			message = strings.TrimSpace(message[idx+1:])
		}
	}
	message = strings.TrimSpace(message)

	// Create context around the error line
	context := ""
	if line > 0 {
		context = extractContext(content, line, 2)
	}

	// Generate suggestion based on error type
	suggestion := generateSuggestion(message, content, line)

	return &ParseError{
		Line:       line,
		Column:     column,
		Message:    message,
		Context:    context,
		Suggestion: suggestion,
	}
}

// extractContext returns lines around the error for context
func extractContext(content string, lineNum int, contextLines int) string {
	lines := strings.Split(content, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return ""
	}

	start := lineNum - contextLines - 1
	if start < 0 {
		start = 0
	}

	end := lineNum + contextLines
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		prefix := "  "
		if i == lineNum-1 {
			prefix = "→ " // Mark the error line
		}
		sb.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, i+1, lines[i]))
	}

	return sb.String()
}

// generateSuggestion provides context-aware suggestions for fixing common errors
func generateSuggestion(message, content string, line int) string {
	messageLower := strings.ToLower(message)

	// Common error patterns and suggestions
	switch {
	case strings.Contains(messageLower, "mapping values are not allowed"):
		return "Check for incorrect indentation or missing colon. YAML requires consistent indentation (use spaces, not tabs) and colons for key-value pairs."

	case strings.Contains(messageLower, "did not find expected key"):
		return "Check for missing or incorrect key name. Ensure all mapping keys are properly quoted if they contain special characters."

	case strings.Contains(messageLower, "block end"):
		return "Check for incorrect indentation or unclosed block. Ensure consistent spacing (2 or 4 spaces) throughout the file."

	case strings.Contains(messageLower, "unmarshal"):
		if strings.Contains(messageLower, "into") {
			// Type mismatch
			return "Value type doesn't match expected type. Check the documentation for the correct data type (string, number, boolean, array, or object)."
		}
		return "Failed to parse value. Check that the value format matches the expected type."

	case strings.Contains(messageLower, "cannot unmarshal"):
		return "Type mismatch detected. Verify the field expects the type of value you're providing (e.g., string vs number vs boolean)."

	case strings.Contains(messageLower, "duplicate") || strings.Contains(messageLower, "already defined"):
		return "Duplicate key detected. Remove one of the duplicate entries or rename one of them."

	case strings.Contains(messageLower, "invalid"):
		return "Invalid value or format. Check the documentation for valid values and proper formatting."

	case strings.Contains(messageLower, "unknown field") || strings.Contains(messageLower, "field") && strings.Contains(messageLower, "not found"):
		return "Field name not recognized. Check for typos or refer to the documentation for valid field names."

	case strings.Contains(messageLower, "required"):
		return "Required field is missing. Add the missing field with an appropriate value."

	case strings.Contains(messageLower, "tab"):
		return "Tabs detected. YAML requires spaces for indentation, not tabs. Replace all tabs with spaces (2 or 4 spaces per level)."

	case strings.Contains(messageLower, "indent"):
		return "Incorrect indentation. Ensure consistent spacing throughout the file (typically 2 or 4 spaces per indentation level)."

	case strings.Contains(messageLower, "anchor") || strings.Contains(messageLower, "alias"):
		return "Issue with YAML anchor or alias. Ensure anchors are defined with '&' before being referenced with '*'."

	case strings.Contains(messageLower, "eof") || strings.Contains(messageLower, "end of file"):
		return "Unexpected end of file. Check for unclosed quotes, brackets, or incomplete structures."

	default:
		// Check line content for common issues
		if line > 0 {
			lines := strings.Split(content, "\n")
			if line <= len(lines) {
				lineContent := lines[line-1]

				// Check for tabs
				if strings.Contains(lineContent, "\t") {
					return "Line contains tabs. Replace tabs with spaces for proper YAML indentation."
				}

				// Check for trailing spaces on key
				if strings.Contains(lineContent, ": ") && strings.HasSuffix(strings.Split(lineContent, ":")[0], " ") {
					return "Remove trailing spaces before the colon in the key name."
				}
			}
		}

		return "Review the YAML syntax around this line. Common issues: incorrect indentation, missing colons, or unquoted special characters."
	}
}

// Validate checks if YAML content is valid without unmarshaling into a specific type
func Validate(data []byte) error {
	var v interface{}
	return Unmarshal(data, &v)
}

// UnmarshalStrict parses YAML with strict mode (rejects unknown fields)
func UnmarshalStrict(data []byte, v interface{}) error {
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Strict mode

	err := decoder.Decode(v)
	if err == nil {
		return nil
	}

	return enhanceError(err, string(data))
}
