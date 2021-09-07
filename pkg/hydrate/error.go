package hydrate

import "github.com/google/nomos/pkg/status"

// HydrationError is a wrapper of the error in the hydration process with the error code.
type HydrationError interface {
	// Code is the error code to indicate if it is a user error or an internal error.
	Code() string
	error
}

// ActionableError represents the user actionable hydration error.
type ActionableError struct {
	error
}

// NewActionableError returns the wrapper of the user actionable error.
func NewActionableError(e error) ActionableError {
	return ActionableError{e}
}

// Code returns the user actionable error code.
func (e ActionableError) Code() string {
	return status.ActionableHydrationErrorCode
}

// InternalError represents the internal hydration error.
type InternalError struct {
	error
}

// NewInternalError returns the wrapper of the internal error.
func NewInternalError(e error) InternalError {
	return InternalError{e}
}

// Code returns the internal error code.
func (e InternalError) Code() string {
	return status.InternalHydrationErrorCode
}

// HydrationErrorPayload is the payload of the hydration error in the error file.
type HydrationErrorPayload struct {
	// Code is the error code to indicate if it is a user error or an internal error.
	Code string
	// Error is the message of the hydration error.
	Error string
}
