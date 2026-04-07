package utils

import "errors"

// Common errors used throughout the LangGraph compatibility layer.
var (
	ErrInvalidFilename = errors.New("invalid filename")
	ErrPathTraversal   = errors.New("path traversal detected")
	ErrFileTooLarge    = errors.New("file too large")
	ErrInvalidArchive  = errors.New("invalid archive")
	ErrSkillNotFound   = errors.New("skill not found")
	ErrAgentNotFound   = errors.New("agent not found")
	ErrThreadNotFound  = errors.New("thread not found")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInternal        = errors.New("internal error")
)
