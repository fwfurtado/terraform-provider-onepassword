package client

import (
	"fmt"
)

type OnePasswordErrorCode string

const (
	OnePasswordErrorNotFound OnePasswordErrorCode = "404"
)

type OnePasswordError struct {
	Message string
	Code    OnePasswordErrorCode
}

func (e *OnePasswordError) Error() string {
	return fmt.Sprintf("1password error [%s]: %s", e.Code, e.Message)
}

func newOnePasswordError(message string, code OnePasswordErrorCode) *OnePasswordError {
	return &OnePasswordError{
		Message: message,
		Code:    code,
	}
}

func ErrFieldNotFound(item, field, section string) *OnePasswordError {
	if section != "" {
		return newOnePasswordError(fmt.Sprintf("field '%s' not found in section '%s' of item '%s'", field, section, item), OnePasswordErrorNotFound)
	}

	return newOnePasswordError(fmt.Sprintf("field '%s' not found in item '%s'", field, item), OnePasswordErrorNotFound)
}

func ErrSectionNotFound(item, section string) *OnePasswordError {
	return newOnePasswordError(fmt.Sprintf("section '%s' not found in item '%s'", section, item), OnePasswordErrorNotFound)
}
