package apperror

import "net/http"

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message, nil)
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message, nil)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, message, nil)
}

func NotFound(message string) *AppError {
	return New(http.StatusNotFound, message, nil)
}

func Internal(err error) *AppError {
	return New(http.StatusInternalServerError, "Internal Server Error", err)
}
