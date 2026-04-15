package models

import (
	"encoding/json"
	"testing"
)

func TestChatCompletionRequest(t *testing.T) {
	jsonStr := `{
		"model": "gpt-3.5-turbo",
		"messages": [
			{"role": "user", "content": "Hello"}
		],
		"temperature": 0.7,
		"stream": false
	}`

	var req ChatCompletionRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model gpt-3.5-turbo, got %s", req.Model)
	}

	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}

	if req.Temperature == nil || *req.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %v", req.Temperature)
	}
}

func TestChatCompletionResponse(t *testing.T) {
	content := "Hello, how can I help you?"
	resp := NewChatCompletionResponse(
		"gpt-3.5-turbo",
		[]ChatCompletionChoice{
			{
				Index: 0,
				Message: &Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		&Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	)

	if resp.Object != "chat.completion" {
		t.Errorf("Expected object chat.completion, got %s", resp.Object)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != content {
		t.Errorf("Expected content %s, got %s", content, resp.Choices[0].Message.Content)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("Response JSON: %s", string(data))
}

func TestStreamResponse(t *testing.T) {
	content := "Hello"
	resp := NewStreamResponse(
		"gpt-3.5-turbo",
		[]ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: &Message{
					Role:    "assistant",
					Content: content,
				},
			},
		},
	)

	if resp.Object != "chat.completion.chunk" {
		t.Errorf("Expected object chat.completion.chunk, got %s", resp.Object)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("Stream Response JSON: %s", string(data))
}
