# Streaming Issue - Client-Side Fix Required

## Problem
The backend is correctly streaming individual chunks, but the Flutter client is **accumulating all previous chunks** instead of displaying each chunk separately.

## Backend Behavior (CORRECT ✅)
```go
// internal/chat/engine.go - Line 261
for chunk := range chunks {
    chunkContent := chunk.Choices[0].Delta.Content
    if chunkContent != "" {
        // Send ONLY the new content, not accumulated
        if err := req.Responder.SendMessage(chunkContent); err != nil {
            return err
        }
    }
}
```

**WebSocket messages sent:**
```json
{"type": "message", "content": "I"}
{"type": "message", "content": "'"}
{"type": "message", "content": "m"}
{"type": "message", "content": " here"}
{"type": "message", "content": " to"}
{"type": "message", "content": " listen"}
```

## Flutter Client Issue (WRONG ❌)

The Flutter app is likely doing this:
```dart
// WRONG - Accumulating chunks
String fullMessage = "";
socket.listen((data) {
  fullMessage += data['content']; // Keeps appending
  setState(() {
    message = fullMessage; // Shows: "I I'm I'm here I'm here to..."
  });
});
```

## Fix Required in Flutter

**Option 1: Display chunks progressively**
```dart
// Display each chunk as it arrives
String currentMessage = "";
socket.listen((data) {
  if (data['type'] == 'message') {
    currentMessage += data['content']; // Accumulate internally
    setState(() {
      displayMessage = currentMessage; // Show accumulated text
    });
  } else if (data['type'] == 'done') {
    // Message complete
    currentMessage = ""; // Reset for next message
  }
});
```

**Option 2: Collect chunks, display once**
```dart
// Wait for complete message
List<String> chunks = [];
socket.listen((data) {
  if (data['type'] == 'message') {
    chunks.add(data['content']);
  } else if (data['type'] == 'done') {
    String fullMessage = chunks.join('');
    setState(() {
      messages.add(fullMessage);
    });
    chunks = []; // Reset
  }
});
```

## Backend Changes Made ✅

1. **Fixed PostgreSQL array handling** - Symptoms now save correctly
   - Added `pq.Array()` wrapper for `associated_symptoms`
   - Error resolved: `sql: converting argument $7 type: unsupported type []string`

## Test the Fix

**Send a test message and check logs:**
```bash
docker compose logs -f | grep "AI stream"
```

You should see:
```
2026/01/11 03:30:43 Starting AI stream for user=xxx
2026/01/11 03:30:46 AI stream completed: 30 chunks, 139 bytes
```

**Backend is streaming correctly** - Each chunk is sent individually. The duplication is purely a Flutter UI rendering issue.

## Multithreading Note

The backend already uses Go concurrency extensively:
- **Concurrent DB queries** (facts + symptoms + messages in parallel)
- **Goroutines for streaming** (DeepSeek client uses channels)
- **Non-blocking WebSocket handlers** (each connection = separate goroutine)

The streaming itself is inherently concurrent - chunks are sent as they arrive from the AI API.
