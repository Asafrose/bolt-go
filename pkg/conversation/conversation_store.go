package conversation

import (
	"errors"
	"sync"
	"time"
)

// ConversationStore defines the interface for conversation storage backends
type ConversationStore interface {
	// Set stores conversation state with optional expiration
	Set(conversationID string, value any, expiresAt *time.Time) error
	// Get retrieves conversation state
	Get(conversationID string) (any, error)
	// Delete removes conversation state
	Delete(conversationID string) error
}

// MemoryStore is the default in-memory implementation of ConversationStore
// This should not be used in situations where there is more than one instance
// of the app running because state will not be shared amongst the processes.
type MemoryStore struct {
	mu    sync.RWMutex
	state map[string]*conversationEntry
}

type conversationEntry struct {
	Value     any
	ExpiresAt *time.Time
}

// NewMemoryStore creates a new in-memory conversation store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		state: make(map[string]*conversationEntry),
	}
}

// Set stores conversation state with optional expiration
func (s *MemoryStore) Set(conversationID string, value any, expiresAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state[conversationID] = &conversationEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	return nil
}

// Get retrieves conversation state
func (s *MemoryStore) Get(conversationID string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.state[conversationID]
	if !exists {
		return nil, errors.New("conversation not found")
	}

	// Check if expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// Clean up expired entry
		delete(s.state, conversationID)
		return nil, errors.New("conversation expired")
	}

	return entry.Value, nil
}

// Delete removes conversation state
func (s *MemoryStore) Delete(conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.state, conversationID)
	return nil
}

// CleanupExpired removes all expired entries
func (s *MemoryStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, entry := range s.state {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(s.state, id)
		}
	}
}
