package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/my-llm-api/models"
)

type SiliconFlowProvider struct {
	baseURL    string
	httpClient HTTPDoer
}

// NewSiliconFlowProvider 创建使用默认共享 HTTP Client 的 Provider
func NewSiliconFlowProvider(baseURL string) *SiliconFlowProvider {
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}
	return &SiliconFlowProvider{
		baseURL:    baseURL,
		httpClient: DefaultHTTPClient, // 复用全局连接池
	}
}

// NewSiliconFlowProviderWithClient 创建使用自定义 HTTP Client 的 Provider（用于测试）
func NewSiliconFlowProviderWithClient(baseURL string, client HTTPDoer) *SiliconFlowProvider {
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}
	return &SiliconFlowProvider{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (p *SiliconFlowProvider) Name() string {
	return "siliconflow"
}

func (p *SiliconFlowProvider) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result models.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *SiliconFlowProvider) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error) {
	streamReq := *req
	streamReq.Stream = true

	body, err := json.Marshal(streamReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	streamChan := make(chan *models.ChatCompletionStreamResponse, 100)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		close(streamChan)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		close(streamChan)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		// Expand the scanner buffer to 1 MB to handle large SSE data lines
		// (e.g., responses containing base64-encoded content or very long tool calls).
		const maxScanBufSize = 1 * 1024 * 1024 // 1 MB
		scanner.Buffer(make([]byte, 64*1024), maxScanBufSize)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				break
			}

			var streamResp models.ChatCompletionStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			select {
			case streamChan <- &streamResp:
			case <-ctx.Done():
				return
			}
		}
	}()

	return streamChan, nil
}

// 确保 SiliconFlowProvider 实现了 HTTPDoer 接口
var _ HTTPDoer = (*http.Client)(nil)

// MockSiliconFlowProvider 用于测试的 Mock Provider
type MockSiliconFlowProvider struct {
	ChatCompletionFunc       func(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error)
	ChatCompletionStreamFunc func(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error)
}

func (m *MockSiliconFlowProvider) Name() string {
	return "mock"
}

func (m *MockSiliconFlowProvider) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error) {
	if m.ChatCompletionFunc != nil {
		return m.ChatCompletionFunc(ctx, req, apiKey)
	}
	return nil, fmt.Errorf("ChatCompletionFunc not implemented")
}

func (m *MockSiliconFlowProvider) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error) {
	if m.ChatCompletionStreamFunc != nil {
		return m.ChatCompletionStreamFunc(ctx, req, apiKey)
	}
	ch := make(chan *models.ChatCompletionStreamResponse)
	close(ch)
	return ch, fmt.Errorf("ChatCompletionStreamFunc not implemented")
}

// 验证接口实现
var _ Provider = (*SiliconFlowProvider)(nil)
var _ Provider = (*MockSiliconFlowProvider)(nil)

// sleep 短暂休眠（用于重试）
func sleep(d time.Duration) {
	time.Sleep(d)
}
