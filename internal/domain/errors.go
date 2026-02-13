package domain

import "fmt"

// ErrorCategory classifies errors for logging and response.
type ErrorCategory string

const (
	ErrCatValidation ErrorCategory = "validation"
	ErrCatRateLimit  ErrorCategory = "rate_limit"
	ErrCatVertexErr  ErrorCategory = "vertex_error"
	ErrCatUnknown    ErrorCategory = "unknown"
)

// AppError wraps an error with a category and HTTP status code.
type AppError struct {
	Category   ErrorCategory
	Message    string
	StatusCode int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewValidationError(msg string) *AppError {
	return &AppError{
		Category:   ErrCatValidation,
		Message:    msg,
		StatusCode: 400,
	}
}

func NewRateLimitError() *AppError {
	return &AppError{
		Category:   ErrCatRateLimit,
		Message:    "rate limit exceeded",
		StatusCode: 429,
	}
}

func NewVertexError(msg string, err error) *AppError {
	return &AppError{
		Category:   ErrCatVertexErr,
		Message:    msg,
		StatusCode: 502,
		Err:        err,
	}
}

func NewInternalError(msg string, err error) *AppError {
	return &AppError{
		Category:   ErrCatUnknown,
		Message:    msg,
		StatusCode: 500,
		Err:        err,
	}
}
