package memory

import (
	"sync"
	"time"
)

// Message represents a chat message in short-term memory
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// UserFact represents a long-term memory fact about a user
type UserFact struct {
	Key        string    `json:"key"` // e.g., "pregnancy_week", "diet", "exercise"
	Value      string    `json:"value"`
	Confidence float64   `json:"confidence"` // 0.0 to 1.0
	UpdatedAt  time.Time `json:"updated_at"`
}

// UserMemory holds both short-term and long-term memory for a user
type UserMemory struct {
	ShortTerm []Message           // Last N messages
	Facts     map[string]UserFact // Long-term facts
	mu        sync.RWMutex
}

// MemoryManager manages memory for all users
type MemoryManager struct {
	users               map[string]*UserMemory
	shortTermMemorySize int
	mu                  sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(shortTermMemorySize int) *MemoryManager {
	return &MemoryManager{
		users:               make(map[string]*UserMemory),
		shortTermMemorySize: shortTermMemorySize,
	}
}

// AddMessage adds a message to the user's short-term memory
func (m *MemoryManager) AddMessage(userID string, msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create user memory
	userMem, exists := m.users[userID]
	if !exists {
		userMem = &UserMemory{
			ShortTerm: make([]Message, 0, m.shortTermMemorySize),
			Facts:     make(map[string]UserFact),
		}
		m.users[userID] = userMem
	}

	userMem.mu.Lock()
	defer userMem.mu.Unlock()

	// Add message to short-term memory
	userMem.ShortTerm = append(userMem.ShortTerm, msg)

	// Enforce size limit (keep only the most recent N messages)
	if len(userMem.ShortTerm) > m.shortTermMemorySize {
		// Remove oldest messages
		userMem.ShortTerm = userMem.ShortTerm[len(userMem.ShortTerm)-m.shortTermMemorySize:]
	}
}

// GetShortTermMemory retrieves the user's short-term memory
func (m *MemoryManager) GetShortTermMemory(userID string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userMem, exists := m.users[userID]
	if !exists {
		return []Message{}
	}

	userMem.mu.RLock()
	defer userMem.mu.RUnlock()

	// Return a copy to avoid external mutation
	history := make([]Message, len(userMem.ShortTerm))
	copy(history, userMem.ShortTerm)
	return history
}

// AddFact adds or updates a fact in the user's long-term memory
func (m *MemoryManager) AddFact(userID string, fact UserFact) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create user memory
	userMem, exists := m.users[userID]
	if !exists {
		userMem = &UserMemory{
			ShortTerm: make([]Message, 0, m.shortTermMemorySize),
			Facts:     make(map[string]UserFact),
		}
		m.users[userID] = userMem
	}

	userMem.mu.Lock()
	defer userMem.mu.Unlock()

	// Only update if:
	// 1. Fact doesn't exist, OR
	// 2. New confidence is higher than existing
	existingFact, exists := userMem.Facts[fact.Key]
	if !exists || fact.Confidence > existingFact.Confidence {
		userMem.Facts[fact.Key] = fact
	}
}

// GetFacts retrieves all facts for a user
func (m *MemoryManager) GetFacts(userID string) []UserFact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userMem, exists := m.users[userID]
	if !exists {
		return []UserFact{}
	}

	userMem.mu.RLock()
	defer userMem.mu.RUnlock()

	// Convert map to slice
	facts := make([]UserFact, 0, len(userMem.Facts))
	for _, fact := range userMem.Facts {
		facts = append(facts, fact)
	}
	return facts
}

// GetFactByKey retrieves a specific fact by key
func (m *MemoryManager) GetFactByKey(userID, key string) (UserFact, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userMem, exists := m.users[userID]
	if !exists {
		return UserFact{}, false
	}

	userMem.mu.RLock()
	defer userMem.mu.RUnlock()

	fact, exists := userMem.Facts[key]
	return fact, exists
}

// ClearShortTermMemory clears the user's short-term memory
func (m *MemoryManager) ClearShortTermMemory(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userMem, exists := m.users[userID]
	if !exists {
		return
	}

	userMem.mu.Lock()
	defer userMem.mu.Unlock()

	userMem.ShortTerm = make([]Message, 0, m.shortTermMemorySize)
}

// RemoveFact removes a specific fact from long-term memory
func (m *MemoryManager) RemoveFact(userID, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userMem, exists := m.users[userID]
	if !exists {
		return
	}

	userMem.mu.Lock()
	defer userMem.mu.Unlock()

	delete(userMem.Facts, key)
}
