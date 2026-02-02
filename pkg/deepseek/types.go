package deepseek

import (
    "github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

// Alias types for backward compatibility or convenience if needed,
// but for this refactor we will switch to using llm.* types directly.
// Keeping this file for potential DeepSeek-specific types in the future.

type ChatMessage = llm.ChatMessage
type ChatRequest = llm.ChatRequest
type ChatChunk = llm.ChatChunk
type Delta = llm.Delta
type ChatResponse = llm.ChatResponse
