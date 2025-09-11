package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// MemoryInstallationStore is an in-memory implementation of InstallationStore
// This should only be used for development/testing - use a persistent store in production
type MemoryInstallationStore struct {
	installations map[string]*Installation
	mutex         sync.RWMutex
}

// NewMemoryInstallationStore creates a new in-memory installation store
func NewMemoryInstallationStore() *MemoryInstallationStore {
	return &MemoryInstallationStore{
		installations: make(map[string]*Installation),
	}
}

// StoreInstallation stores an installation in memory
func (m *MemoryInstallationStore) StoreInstallation(ctx context.Context, installation *Installation) error {
	if installation == nil {
		return errors.New("installation cannot be nil")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate key based on installation
	key := m.generateKey(installation)
	m.installations[key] = installation

	return nil
}

// FetchInstallation retrieves an installation from memory
func (m *MemoryInstallationStore) FetchInstallation(ctx context.Context, query InstallationQuery) (*Installation, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Generate key based on query
	key := m.generateKeyFromQuery(query)
	installation, exists := m.installations[key]
	if !exists {
		return nil, fmt.Errorf("installation not found for query: %+v", query)
	}

	return installation, nil
}

// DeleteInstallation removes an installation from memory
func (m *MemoryInstallationStore) DeleteInstallation(ctx context.Context, query InstallationQuery) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate key based on query
	key := m.generateKeyFromQuery(query)
	delete(m.installations, key)

	return nil
}

// generateKey generates a storage key from an installation
func (m *MemoryInstallationStore) generateKey(installation *Installation) string {
	if installation.IsEnterpriseInstall && installation.Enterprise != nil {
		return "enterprise:" + installation.Enterprise.ID
	}
	if installation.Team != nil {
		return "team:" + installation.Team.ID
	}
	// Fallback to app ID if available
	if installation.AppID != "" {
		return "app:" + installation.AppID
	}
	return "unknown"
}

// generateKeyFromQuery generates a storage key from a query
func (m *MemoryInstallationStore) generateKeyFromQuery(query InstallationQuery) string {
	if query.IsEnterpriseInstall && query.EnterpriseID != "" {
		return "enterprise:" + query.EnterpriseID
	}
	if query.TeamID != "" {
		return "team:" + query.TeamID
	}
	return "unknown"
}

// ListInstallations returns all stored installations (for debugging/testing)
func (m *MemoryInstallationStore) ListInstallations(ctx context.Context) map[string]*Installation {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*Installation)
	for key, installation := range m.installations {
		result[key] = installation
	}
	return result
}

// Clear removes all installations (for testing)
func (m *MemoryInstallationStore) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.installations = make(map[string]*Installation)
}
