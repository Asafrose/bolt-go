package oauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ClearStateStore is an in-memory implementation of StateStore
// This should only be used for development/testing - use a persistent store in production
type ClearStateStore struct {
	states map[string]*stateData
	mutex  sync.RWMutex
}

type stateData struct {
	InstallOptions *InstallURLOptions
	ExpiresAt      time.Time
}

// NewClearStateStore creates a new in-memory state store
func NewClearStateStore() *ClearStateStore {
	store := &ClearStateStore{
		states: make(map[string]*stateData),
	}

	// Start cleanup goroutine
	go store.cleanupExpiredStates()

	return store
}

// GenerateStateParam generates a state parameter for OAuth flow
func (c *ClearStateStore) GenerateStateParam(ctx context.Context, installOptions *InstallURLOptions) (string, error) {
	// Generate random state
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	state := hex.EncodeToString(bytes)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Store state with expiration (10 minutes)
	c.states[state] = &stateData{
		InstallOptions: installOptions,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}

	return state, nil
}

// VerifyStateParam verifies a state parameter and returns the associated install options
func (c *ClearStateStore) VerifyStateParam(ctx context.Context, state string) (*InstallURLOptions, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	data, exists := c.states[state]
	if !exists {
		return nil, errors.New("invalid or expired state parameter")
	}

	// Check if expired
	if time.Now().After(data.ExpiresAt) {
		return nil, errors.New("state parameter has expired")
	}

	return data.InstallOptions, nil
}

// cleanupExpiredStates periodically removes expired state entries
func (c *ClearStateStore) cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for state, data := range c.states {
			if now.After(data.ExpiresAt) {
				delete(c.states, state)
			}
		}
		c.mutex.Unlock()
	}
}

// Clear removes all state entries (for testing)
func (c *ClearStateStore) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.states = make(map[string]*stateData)
}

// EncryptedStateStore is a state store that encrypts the install options in the state parameter
type EncryptedStateStore struct {
	secret string
}

// NewEncryptedStateStore creates a new encrypted state store
func NewEncryptedStateStore(secret string) *EncryptedStateStore {
	return &EncryptedStateStore{
		secret: secret,
	}
}

// GenerateStateParam generates an encrypted state parameter
func (e *EncryptedStateStore) GenerateStateParam(ctx context.Context, installOptions *InstallURLOptions) (string, error) {
	// Serialize install options
	data, err := json.Marshal(installOptions)
	if err != nil {
		return "", fmt.Errorf("failed to marshal install options: %w", err)
	}

	// Add timestamp to prevent replay attacks
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%d:%s", timestamp, string(data))

	// Encrypt using AES-GCM
	encrypted, err := e.encrypt([]byte(payload))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt state: %w", err)
	}

	return hex.EncodeToString(encrypted), nil
}

// encrypt encrypts data using AES-GCM
func (e *EncryptedStateStore) encrypt(data []byte) ([]byte, error) {
	// Create a 32-byte key from the secret
	hash := sha256.Sum256([]byte(e.secret))
	key := hash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (e *EncryptedStateStore) decrypt(data []byte) ([]byte, error) {
	// Create a 32-byte key from the secret
	hash := sha256.Sum256([]byte(e.secret))
	key := hash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// VerifyStateParam verifies and decrypts a state parameter
func (e *EncryptedStateStore) VerifyStateParam(ctx context.Context, state string) (*InstallURLOptions, error) {
	// Decode the hex-encoded state
	encryptedData, err := hex.DecodeString(state)
	if err != nil {
		return nil, fmt.Errorf("invalid state parameter format: %w", err)
	}

	// Decrypt the state
	decryptedData, err := e.decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt state: %w", err)
	}

	stateStr := string(decryptedData)

	// Parse timestamp and data
	parts := strings.SplitN(stateStr, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid state parameter structure")
	}

	var timestamp int64
	if _, err := fmt.Sscanf(parts[0], "%d", &timestamp); err != nil {
		return nil, fmt.Errorf("invalid timestamp in state parameter: %w", err)
	}

	// Check if expired (10 minutes)
	if time.Now().Unix()-timestamp > 600 {
		return nil, errors.New("state parameter has expired")
	}

	// Parse the install options
	var installOptions InstallURLOptions
	if err := json.Unmarshal([]byte(parts[1]), &installOptions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal install options: %w", err)
	}

	return &installOptions, nil
}
