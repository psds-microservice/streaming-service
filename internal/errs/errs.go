package errs

import "errors"

// Доменные сентинель-ошибки для маппинга в HTTP коды в handlers.
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrTooManyOperators = errors.New("session has maximum operators")
)
