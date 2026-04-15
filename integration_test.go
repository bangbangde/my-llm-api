//go:build integration

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func TestIntegrationHealth(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Health response: %s", string(body))
}

func TestIntegrationListModels(t *testing.T) {
	resp, err := http.Get(baseURL + "/v1/models")
	if err != nil {
		t.Fatalf("Failed to call models endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Models response: %s", string(body))
}

func TestIntegrationChatCompletion(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "Qwen/Qwen2.5-7B-Instruct",
		"messages": []map[string]string{
			{"role": "user", "content": "你好，请用一句话介绍自己"},
		},
		"max_tokens": 100,
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(baseURL+"/v1/chat/completions", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("Failed to call chat completions endpoint: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Chat completion response: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
		t.Errorf("Response body: %s", string(body))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["object"] != "chat.completion" {
		t.Errorf("Expected object chat.completion, got %v", result["object"])
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Error("Expected non-empty choices array")
	}
}

func TestIntegrationChatCompletionStream(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "Qwen/Qwen2.5-7B-Instruct",
		"messages": []map[string]string{
			{"role": "user", "content": "数到5"},
		},
		"stream":     true,
		"max_tokens": 50,
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(baseURL+"/v1/chat/completions", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("Failed to call chat completions endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
		t.Errorf("Response body: %s", string(body))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	chunkCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		
		if data == "[DONE]" {
			t.Logf("Stream completed")
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			t.Logf("Failed to parse chunk: %v", err)
			continue
		}

		chunkCount++
		t.Logf("Chunk %d: %v", chunkCount, chunk)
		
		if chunkCount > 20 {
			t.Logf("Received enough chunks, stopping...")
			break
		}
	}

	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk")
	}
}

func TestIntegrationOpenAISDKCompatibility(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "Qwen/Qwen2.5-7B-Instruct",
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个有帮助的助手"},
			{"role": "user", "content": "1+1等于几？"},
		},
		"temperature": 0.7,
		"max_tokens":  50,
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call chat completions endpoint: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
		t.Errorf("Response body: %s", string(body))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	requiredFields := []string{"id", "object", "created", "model", "choices"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("OpenAI SDK compatibility test passed")
	t.Logf("Response: %s", string(body))
}

func TestIntegrationErrorHandling(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "non-existent-model",
		"messages": []map[string]string{
			{"role": "user", "content": "test"},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(baseURL+"/v1/chat/completions", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("Failed to call chat completions endpoint: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Error response: %s", string(body))
}
