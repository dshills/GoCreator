package fsops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

// Logger defines the interface for logging file operations
type Logger interface {
	// LogFileOperation logs a file operation
	LogFileOperation(ctx context.Context, log models.FileOperationLog) error

	// LogDecision logs a decision made during execution
	LogDecision(ctx context.Context, log models.DecisionLog) error

	// LogError logs an error
	LogError(ctx context.Context, component, operation, message string, err error) error

	// LogInfo logs an informational message
	LogInfo(ctx context.Context, component, operation, message string) error

	// Close closes the logger and flushes any pending writes
	Close() error
}

// FileLogger implements Logger by writing to a JSONL file
type FileLogger struct {
	filePath string
	file     *os.File
	encoder  *json.Encoder
	mu       sync.Mutex
}

// NewFileLogger creates a new file-based logger
func NewFileLogger(logDir string) (*FileLogger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file path
	logPath := filepath.Join(logDir, "file_operations.jsonl")

	// Open file in append mode
	//nolint:gosec // G304: Opening log file - required for file operation logging
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		filePath: logPath,
		file:     file,
		encoder:  json.NewEncoder(file),
	}, nil
}

// LogFileOperation logs a file operation to the JSONL file
func (l *FileLogger) LogFileOperation(ctx context.Context, log models.FileOperationLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure timestamp is set
	if log.LogEntry.Timestamp.IsZero() {
		log.LogEntry.Timestamp = time.Now()
	}

	// Validate the log entry
	if err := log.Validate(); err != nil {
		return fmt.Errorf("invalid file operation log: %w", err)
	}

	// Write to file
	if err := l.encoder.Encode(log); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// LogDecision logs a decision to the JSONL file
func (l *FileLogger) LogDecision(ctx context.Context, log models.DecisionLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure timestamp is set
	if log.LogEntry.Timestamp.IsZero() {
		log.LogEntry.Timestamp = time.Now()
	}

	// Validate the log entry
	if err := log.Validate(); err != nil {
		return fmt.Errorf("invalid decision log: %w", err)
	}

	// Write to file
	if err := l.encoder.Encode(log); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// LogError logs an error to the JSONL file
func (l *FileLogger) LogError(ctx context.Context, component, operation, message string, err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Component: component,
		Operation: operation,
		Message:   message,
		Error:     &errStr,
	}

	if err := l.encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// LogInfo logs an informational message to the JSONL file
func (l *FileLogger) LogInfo(ctx context.Context, component, operation, message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Component: component,
		Operation: operation,
		Message:   message,
	}

	if err := l.encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// Close closes the log file
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// MemoryLogger implements Logger by storing logs in memory (useful for testing)
type MemoryLogger struct {
	entries []interface{}
	mu      sync.Mutex
}

// NewMemoryLogger creates a new in-memory logger
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		entries: make([]interface{}, 0),
	}
}

// LogFileOperation logs a file operation to memory
func (m *MemoryLogger) LogFileOperation(ctx context.Context, log models.FileOperationLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure timestamp is set
	if log.LogEntry.Timestamp.IsZero() {
		log.LogEntry.Timestamp = time.Now()
	}

	// Validate the log entry
	if err := log.Validate(); err != nil {
		return fmt.Errorf("invalid file operation log: %w", err)
	}

	m.entries = append(m.entries, log)
	return nil
}

// LogDecision logs a decision to memory
func (m *MemoryLogger) LogDecision(ctx context.Context, log models.DecisionLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure timestamp is set
	if log.LogEntry.Timestamp.IsZero() {
		log.LogEntry.Timestamp = time.Now()
	}

	// Validate the log entry
	if err := log.Validate(); err != nil {
		return fmt.Errorf("invalid decision log: %w", err)
	}

	m.entries = append(m.entries, log)
	return nil
}

// LogError logs an error to memory
func (m *MemoryLogger) LogError(ctx context.Context, component, operation, message string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Component: component,
		Operation: operation,
		Message:   message,
		Error:     &errStr,
	}

	m.entries = append(m.entries, entry)
	return nil
}

// LogInfo logs an informational message to memory
func (m *MemoryLogger) LogInfo(ctx context.Context, component, operation, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Component: component,
		Operation: operation,
		Message:   message,
	}

	m.entries = append(m.entries, entry)
	return nil
}

// GetEntries returns all logged entries
func (m *MemoryLogger) GetEntries() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent race conditions
	entries := make([]interface{}, len(m.entries))
	copy(entries, m.entries)
	return entries
}

// GetFileOperations returns only file operation logs
func (m *MemoryLogger) GetFileOperations() []models.FileOperationLog {
	m.mu.Lock()
	defer m.mu.Unlock()

	var ops []models.FileOperationLog
	for _, entry := range m.entries {
		if op, ok := entry.(models.FileOperationLog); ok {
			ops = append(ops, op)
		}
	}
	return ops
}

// Clear clears all logged entries
func (m *MemoryLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make([]interface{}, 0)
}

// Close is a no-op for memory logger
func (m *MemoryLogger) Close() error {
	return nil
}

// noopLogger is a logger that does nothing (used when no logger is configured)
type noopLogger struct{}

func (n *noopLogger) LogFileOperation(ctx context.Context, log models.FileOperationLog) error {
	return nil
}

func (n *noopLogger) LogDecision(ctx context.Context, log models.DecisionLog) error {
	return nil
}

func (n *noopLogger) LogError(ctx context.Context, component, operation, message string, err error) error {
	return nil
}

func (n *noopLogger) LogInfo(ctx context.Context, component, operation, message string) error {
	return nil
}

func (n *noopLogger) Close() error {
	return nil
}
