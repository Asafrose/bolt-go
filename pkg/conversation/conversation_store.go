package conversation

import (
	"errors"
	"sync"
	"time"
)

// ConversationStore defines the interface for conversation storage backends
type ConversationStore[ConversationState any] interface {
	// Set stores conversation state with optional expiration
	Set(conversationID string, value ConversationState, expiresAt *time.Time) error
	// Get retrieves conversation state
	Get(conversationID string) (ConversationState, error)
	// Delete removes conversation state
	Delete(conversationID string) error
}

// MemoryStore is the default in-memory implementation of ConversationStore
// This should not be used in situations where there is more than one instance
// of the app running because state will not be shared amongst the processes.
type MemoryStore[ConversationState any] struct {
	mu    sync.RWMutex
	state map[string]*conversationEntry[ConversationState]
}

type conversationEntry[ConversationState any] struct {
	Value     ConversationState
	ExpiresAt *time.Time
}

// NewMemoryStore creates a new in-memory conversation store
func NewMemoryStore[ConversationState any]() *MemoryStore[ConversationState] {
	return &MemoryStore[ConversationState]{
		state: make(map[string]*conversationEntry[ConversationState]),
	}
}

// Set stores conversation state with optional expiration
func (s *MemoryStore[ConversationState]) Set(conversationID string, value ConversationState, expiresAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state[conversationID] = &conversationEntry[ConversationState]{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	return nil
}

// Get retrieves conversation state
func (s *MemoryStore[ConversationState]) Get(conversationID string) (ConversationState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zero ConversationState

	entry, exists := s.state[conversationID]
	if !exists {
		return zero, errors.New("conversation not found")
	}

	// Check if expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// Clean up expired entry
		delete(s.state, conversationID)
		return zero, errors.New("conversation expired")
	}

	return entry.Value, nil
}

// Delete removes conversation state
func (s *MemoryStore[ConversationState]) Delete(conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.state, conversationID)
	return nil
}

// CleanupExpired removes all expired entries
func (s *MemoryStore[ConversationState]) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, entry := range s.state {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(s.state, id)
		}
	}
}
