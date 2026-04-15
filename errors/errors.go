package errors

import (
	"errors"
	"fmt"
)

// 业务错误码
const (
	ErrCodeProviderNotFound = "provider_not_found"
	ErrCodeModelNotFound    = "model_not_found"
	ErrCodeNoAccount        = "no_available_account"
	ErrCodeUpstreamError    = "upstream_error"
	ErrCodeInvalidRequest   = "invalid_request"
	ErrCodeInternalError    = "internal_error"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeAuthFailed       = "authentication_failed"
)

// AppError 应用层错误
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// WrapError 包装错误为应用错误
func WrapError(code, message string, err error) error {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewAppError 创建应用错误
func NewAppError(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case ErrCodeUpstreamError:
			return true
		case ErrCodeRateLimited:
			return true
		default:
			return false
		}
	}

	// 非 AppError 默认可重试（可能是网络错误）
	return true
}

// IsNotFound 判断是否为"未找到"错误
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case ErrCodeProviderNotFound, ErrCodeModelNotFound, ErrCodeNoAccount:
			return true
		}
	}
	return false
}

// FormatError 格式化错误为用户友好的消息
func FormatError(err error) string {
	if err == nil {
		return "unknown error"
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}

	return err.Error()
}

// ErrorResponse 错误响应结构（用于 API 返回）
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// FromAppError 从 AppError 转换为 ErrorDetail
func FromAppError(err error) *ErrorDetail {
	if err == nil {
		return &ErrorDetail{
			Message: "unknown error",
			Type:    "internal_error",
			Code:    ErrCodeInternalError,
		}
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return &ErrorDetail{
			Message: appErr.Message,
			Type:    mapCodeToType(appErr.Code),
			Code:    appErr.Code,
		}
	}

	return &ErrorDetail{
		Message: err.Error(),
		Type:    "internal_error",
		Code:    ErrCodeInternalError,
	}
}

func mapCodeToType(code string) string {
	switch code {
	case ErrCodeInvalidRequest, ErrCodeModelNotFound:
		return "invalid_request_error"
	case ErrCodeAuthFailed:
		return "authentication_error"
	case ErrCodeRateLimited:
		return "rate_limit_error"
	case ErrCodeProviderNotFound, ErrCodeNoAccount:
		return "invalid_request_error"
	default:
		return "internal_error"
	}
}

// AggregateError 聚合多个错误
type AggregateError struct {
	Errors []error
}

func (e *AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors occurred", len(e.Errors))
}

func (e *AggregateError) Unwrap() []error {
	return e.Errors
}

// NewAggregateError 创建聚合错误
func NewAggregateError(errs ...error) *AggregateError {
	return &AggregateError{Errors: errs}
}
