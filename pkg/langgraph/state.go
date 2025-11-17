package langgraph

import (
	"encoding/json"
	"fmt"
	"sync"
)

// State represents the typed state container for a graph execution
// It provides thread-safe access to state data with type preservation
type State interface {
	// Get retrieves a value from the state by key
	Get(key string) (interface{}, bool)

	// Set stores a value in the state by key
	Set(key string, value interface{})

	// Delete removes a value from the state
	Delete(key string)

	// Keys returns all keys in the state
	Keys() []string

	// Clone creates a deep copy of the state
	Clone() (State, error)

	// ToJSON serializes the state to JSON
	ToJSON() ([]byte, error)

	// FromJSON deserializes the state from JSON
	FromJSON(data []byte) error
}

// MapState is a map-based implementation of State with thread-safe operations
type MapState struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewMapState creates a new MapState instance
func NewMapState() *MapState {
	return &MapState{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value from the state
func (s *MapState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Set stores a value in the state
func (s *MapState) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Delete removes a value from the state
func (s *MapState) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Keys returns all keys in the state
func (s *MapState) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Clone creates a deep copy of the state
func (s *MapState) Clone() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Serialize current state
	data, err := json.Marshal(s.data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	// Create new state and deserialize
	newState := NewMapState()
	if err := json.Unmarshal(data, &newState.data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return newState, nil
}

// ToJSON serializes the state to JSON
func (s *MapState) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(s.data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}
	return data, nil
}

// FromJSON deserializes the state from JSON
func (s *MapState) FromJSON(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := json.Unmarshal(data, &s.data); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return nil
}

// GetString retrieves a string value from the state
func (s *MapState) GetString(key string) (string, bool) {
	val, ok := s.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an int value from the state
func (s *MapState) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}

	// Handle both int and float64 (JSON unmarshaling default)
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a bool value from the state
func (s *MapState) GetBool(key string) (bool, bool) {
	val, ok := s.Get(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// GetSlice retrieves a slice value from the state
func (s *MapState) GetSlice(key string) ([]interface{}, bool) {
	val, ok := s.Get(key)
	if !ok {
		return nil, false
	}
	slice, ok := val.([]interface{})
	return slice, ok
}

// GetMap retrieves a map value from the state
func (s *MapState) GetMap(key string) (map[string]interface{}, bool) {
	val, ok := s.Get(key)
	if !ok {
		return nil, false
	}
	m, ok := val.(map[string]interface{})
	return m, ok
}

// Merge merges another state into this state
func (s *MapState) Merge(other State) error {
	otherMap, ok := other.(*MapState)
	if !ok {
		return fmt.Errorf("can only merge MapState instances")
	}

	otherMap.mu.RLock()
	defer otherMap.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range otherMap.data {
		s.data[k] = v
	}

	return nil
}
