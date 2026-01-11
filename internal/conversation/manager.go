package conversation

import (
	"sync"
	"time"
)

// State tracks the current conversation context
type State struct {
	UserID          string
	PrimaryConcern  string    // The main topic user brought up
	FollowUpCount   int       // How many follow-ups on current topic
	LastTopicChange time.Time // When topic last changed
	SecondaryTopics []string  // Side topics mentioned
}

// Manager tracks conversation state per user
type Manager struct {
	mu     sync.RWMutex
	states map[string]*State
}

// NewManager creates a conversation state manager
func NewManager() *Manager {
	return &Manager{
		states: make(map[string]*State),
	}
}

// GetState retrieves or creates conversation state
func (m *Manager) GetState(userID string) *State {
	m.mu.RLock()
	state, exists := m.states[userID]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		state = &State{
			UserID:          userID,
			SecondaryTopics: make([]string, 0),
			LastTopicChange: time.Now(),
		}
		m.states[userID] = state
		m.mu.Unlock()
	}

	return state
}

// SetPrimaryConcern sets the main topic user is asking about
func (m *Manager) SetPrimaryConcern(userID, concern string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.states[userID]
	if state == nil {
		state = &State{
			UserID:          userID,
			SecondaryTopics: make([]string, 0),
		}
		m.states[userID] = state
	}

	state.PrimaryConcern = concern
	state.FollowUpCount = 0
	state.LastTopicChange = time.Now()
}

// IncrementFollowUp increments follow-up count
func (m *Manager) IncrementFollowUp(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.states[userID]; exists {
		state.FollowUpCount++
	}
}

// AddSecondaryTopic tracks side topics mentioned
func (m *Manager) AddSecondaryTopic(userID, topic string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.states[userID]; exists {
		state.SecondaryTopics = append(state.SecondaryTopics, topic)
	}
}

// ShouldRefocus determines if we need to steer back to primary concern
func (m *Manager) ShouldRefocus(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[userID]
	if !exists || state.PrimaryConcern == "" {
		return false
	}

	// Refocus after 2 follow-ups on secondary topics
	return state.FollowUpCount >= 2 && len(state.SecondaryTopics) > 0
}

// Reset clears conversation state (e.g., after primary concern is resolved)
func (m *Manager) Reset(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, userID)
}
