package errs

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrPlanNotFound   = errors.New("plan not found")
	ErrLLMUnavailable = errors.New("llm unavailable")
	ErrInvalidLLMData = errors.New("invalid llm response")
)
