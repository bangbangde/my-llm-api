package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/my-llm-api/models"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries     int           // 最大重试次数
	InitialBackoff time.Duration // 初始退避时间
	MaxBackoff     time.Duration // 最大退避时间
	Jitter         bool          // 是否添加随机抖动
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:     3,
	InitialBackoff: 100 * time.Millisecond,
	MaxBackoff:     2 * time.Second,
	Jitter:         true,
}

// RetryableError 可重试的错误
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError 创建可重试错误
func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 RetryableError
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	// 检查是否是网络错误或上游错误
	errMsg := err.Error()

	// 可重试的错误类型
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"i/o timeout",
		"temporary failure",
		"server misbehaving",
		"upstream error",
		"status 502",
		"status 503",
		"status 504",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errMsg), pattern) {
			return true
		}
	}

	// 4xx 客户端错误不重试（除了 429 rate limit）
	if strings.Contains(errMsg, "status 4") && !strings.Contains(errMsg, "status 429") {
		return false
	}

	return false
}

// withRetry 带指数退避的重试
func withRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := cfg.InitialBackoff * time.Duration(1<<(attempt-1))
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}

			// 添加随机抖动以避免惊群效应
			if cfg.Jitter {
				jitter := time.Duration(float64(backoff) * 0.2 * (float64(attempt%3) - 1))
				backoff += jitter
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// 判断是否可重试
		if !IsRetryable(lastErr) {
			return lastErr
		}
	}

	return lastErr
}

// ChatCompletionWithRetry 带重试的聊天补全
func (s *Scheduler) ChatCompletionWithRetry(ctx context.Context, req *models.ChatCompletionRequest, cfg RetryConfig) (*models.ChatCompletionResponse, error) {
	var lastErr error
	var resp *models.ChatCompletionResponse

	err := withRetry(ctx, cfg, func() error {
		var err error
		resp, err = s.ChatCompletion(ctx, req)
		if err != nil {
			lastErr = err
			if IsRetryable(err) {
				return &RetryableError{Err: err}
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, lastErr
	}

	return resp, nil
}

// ChatCompletionStreamWithRetry 带重试的流式聊天补全
// 注意：流式重试只重试建立连接，stream 建立后不再重试
func (s *Scheduler) ChatCompletionStreamWithRetry(ctx context.Context, req *models.ChatCompletionRequest, cfg RetryConfig) (<-chan *models.ChatCompletionStreamResponse, error) {
	var lastErr error
	var streamChan <-chan *models.ChatCompletionStreamResponse

	err := withRetry(ctx, cfg, func() error {
		var err error
		streamChan, err = s.ChatCompletionStream(ctx, req)
		if err != nil {
			lastErr = err
			if IsRetryable(err) {
				return &RetryableError{Err: err}
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, lastErr
	}

	return streamChan, nil
}

// MultiError 收集多个错误
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	return fmt.Sprintf("%d errors occurred", len(e.Errors))
}

func (e *MultiError) Append(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}
