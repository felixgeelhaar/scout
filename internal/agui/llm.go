package agui

import "context"

// LLMProvider sends messages and tools to an LLM and streams back responses.
type LLMProvider interface {
	StreamChat(ctx context.Context, req ChatRequest, cb func(ChatChunk)) error
}

// ChatRequest is the input to the LLM.
type ChatRequest struct {
	System   string
	Messages []ChatMessage
	Tools    []ToolDef
}

// ChatMessage is a single message in the conversation.
type ChatMessage struct {
	Role       string     `json:"role"` // "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents an LLM tool invocation.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"` // raw JSON string
}

// ChatChunk is a streaming chunk from the LLM.
type ChatChunk struct {
	Type         string // "text", "tool_use_start", "tool_use_delta", "tool_use_end", "stop", "error"
	Text         string
	ToolCallID   string
	ToolName     string
	ArgsFragment string
}
