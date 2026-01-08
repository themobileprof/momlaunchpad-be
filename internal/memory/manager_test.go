package memory

import (
	"testing"
	"time"
)

func TestMemoryManager_AddMessage(t *testing.T) {
	manager := NewMemoryManager(5, nil) // nil DB for testing

	msg := Message{
		Role:      "user",
		Content:   "Hello, how are you?",
		Timestamp: time.Now(),
	}

	manager.AddMessage("user123", msg)

	history := manager.GetShortTermMemory("user123")
	if len(history) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history))
	}

	if history[0].Content != msg.Content {
		t.Errorf("Expected content %q, got %q", msg.Content, history[0].Content)
	}
}

func TestMemoryManager_ShortTermMemoryLimit(t *testing.T) {
	manager := NewMemoryManager(3, nil)

	messages := []Message{
		{Role: "user", Content: "Message 1", Timestamp: time.Now()},
		{Role: "assistant", Content: "Response 1", Timestamp: time.Now()},
		{Role: "user", Content: "Message 2", Timestamp: time.Now()},
		{Role: "assistant", Content: "Response 2", Timestamp: time.Now()},
		{Role: "user", Content: "Message 3", Timestamp: time.Now()},
	}

	for _, msg := range messages {
		manager.AddMessage("user123", msg)
	}

	history := manager.GetShortTermMemory("user123")
	if len(history) != 3 {
		t.Errorf("Expected 3 messages (limit), got %d", len(history))
	}

	// Should keep the most recent 3: "Message 2", "Response 2", "Message 3"
	if history[0].Content != "Message 2" {
		t.Errorf("Expected oldest kept message to be 'Message 2', got %q", history[0].Content)
	}

	if history[1].Content != "Response 2" {
		t.Errorf("Expected middle message to be 'Response 2', got %q", history[1].Content)
	}

	if history[2].Content != "Message 3" {
		t.Errorf("Expected newest message to be 'Message 3', got %q", history[2].Content)
	}
}

func TestMemoryManager_AddFact(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	fact := UserFact{
		Key:        "pregnancy_week",
		Value:      "14",
		Confidence: 0.9,
		UpdatedAt:  time.Now(),
	}

	manager.AddFact("user123", fact)

	facts := manager.GetFacts("user123")
	if len(facts) != 1 {
		t.Errorf("Expected 1 fact, got %d", len(facts))
	}

	if facts[0].Key != "pregnancy_week" {
		t.Errorf("Expected key 'pregnancy_week', got %q", facts[0].Key)
	}
}

func TestMemoryManager_UpdateFact(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	fact1 := UserFact{
		Key:        "pregnancy_week",
		Value:      "14",
		Confidence: 0.8,
		UpdatedAt:  time.Now(),
	}
	manager.AddFact("user123", fact1)

	fact2 := UserFact{
		Key:        "pregnancy_week",
		Value:      "15",
		Confidence: 0.9,
		UpdatedAt:  time.Now().Add(time.Hour),
	}
	manager.AddFact("user123", fact2)

	facts := manager.GetFacts("user123")
	if len(facts) != 1 {
		t.Errorf("Expected 1 fact (updated), got %d", len(facts))
	}

	if facts[0].Value != "15" {
		t.Errorf("Expected updated value '15', got %q", facts[0].Value)
	}

	if facts[0].Confidence != 0.9 {
		t.Errorf("Expected updated confidence 0.9, got %v", facts[0].Confidence)
	}
}

func TestMemoryManager_UpdateFactLowerConfidence(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	fact1 := UserFact{
		Key:        "diet",
		Value:      "vegetarian",
		Confidence: 0.9,
		UpdatedAt:  time.Now(),
	}
	manager.AddFact("user123", fact1)

	fact2 := UserFact{
		Key:        "diet",
		Value:      "vegan",
		Confidence: 0.7,
		UpdatedAt:  time.Now().Add(time.Hour),
	}
	manager.AddFact("user123", fact2)

	facts := manager.GetFacts("user123")
	if len(facts) != 1 {
		t.Errorf("Expected 1 fact, got %d", len(facts))
	}

	if facts[0].Value != "vegetarian" {
		t.Errorf("Expected original value 'vegetarian', got %q", facts[0].Value)
	}
}

func TestMemoryManager_GetFactByKey(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	facts := []UserFact{
		{Key: "pregnancy_week", Value: "14", Confidence: 0.9, UpdatedAt: time.Now()},
		{Key: "diet", Value: "vegetarian", Confidence: 0.8, UpdatedAt: time.Now()},
		{Key: "exercise", Value: "yoga", Confidence: 0.7, UpdatedAt: time.Now()},
	}

	for _, fact := range facts {
		manager.AddFact("user123", fact)
	}

	fact, exists := manager.GetFactByKey("user123", "diet")
	if !exists {
		t.Error("Expected fact to exist")
	}

	if fact.Value != "vegetarian" {
		t.Errorf("Expected value 'vegetarian', got %q", fact.Value)
	}
}

func TestMemoryManager_GetFactByKeyNotFound(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	_, exists := manager.GetFactByKey("user123", "nonexistent")
	if exists {
		t.Error("Expected fact not to exist")
	}
}

func TestMemoryManager_ClearShortTermMemory(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	manager.AddMessage("user123", Message{Role: "user", Content: "Hello", Timestamp: time.Now()})
	manager.AddMessage("user123", Message{Role: "assistant", Content: "Hi", Timestamp: time.Now()})

	manager.ClearShortTermMemory("user123")

	history := manager.GetShortTermMemory("user123")
	if len(history) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(history))
	}
}

func TestMemoryManager_MultipleUsers(t *testing.T) {
	manager := NewMemoryManager(5, nil)

	manager.AddMessage("user1", Message{Role: "user", Content: "User 1 message", Timestamp: time.Now()})
	manager.AddFact("user1", UserFact{Key: "name", Value: "Alice", Confidence: 0.9, UpdatedAt: time.Now()})

	manager.AddMessage("user2", Message{Role: "user", Content: "User 2 message", Timestamp: time.Now()})
	manager.AddFact("user2", UserFact{Key: "name", Value: "Bob", Confidence: 0.9, UpdatedAt: time.Now()})

	history1 := manager.GetShortTermMemory("user1")
	if len(history1) != 1 || history1[0].Content != "User 1 message" {
		t.Error("User 1 history incorrect")
	}

	fact1, _ := manager.GetFactByKey("user1", "name")
	if fact1.Value != "Alice" {
		t.Error("User 1 fact incorrect")
	}

	history2 := manager.GetShortTermMemory("user2")
	if len(history2) != 1 || history2[0].Content != "User 2 message" {
		t.Error("User 2 history incorrect")
	}

	fact2, _ := manager.GetFactByKey("user2", "name")
	if fact2.Value != "Bob" {
		t.Error("User 2 fact incorrect")
	}
}

func TestMemoryManager_ConcurrentAccess(t *testing.T) {
	manager := NewMemoryManager(10, nil)

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			manager.AddMessage("user1", Message{
				Role:      "user",
				Content:   "Message from goroutine 1",
				Timestamp: time.Now(),
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.GetShortTermMemory("user1")
		}
		done <- true
	}()

	<-done
	<-done

	history := manager.GetShortTermMemory("user1")
	if len(history) == 0 {
		t.Error("Expected messages after concurrent access")
	}
}
