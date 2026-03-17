package domain

import "errors"

var (
	ErrNotFound          = errors.New("memory not found")
	ErrOllamaUnavailable = errors.New("ollama is not available — run: ollama serve")
	ErrInvalidRemindExpr = errors.New("could not parse remind expression")
)
