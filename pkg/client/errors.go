package client

import (
	"fmt"
	"net/http"

	"emperror.dev/errors"
)

var (
	ErrNotFound            = errors.New("item not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrBadRequest          = errors.New("bad request")
	ErrInternalServerError = errors.New("internal server error")
	ErrEmptySignature      = errors.New("signature cannot be empty")
	ErrNilData             = errors.New("data cannot be nil")
)

type HTTPError struct {
	StatusCode int
	Status     string
	Message    string
	Body       []byte
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("http error %d (%s): %s", e.StatusCode, e.Status, e.Message)
	}
	return fmt.Sprintf("http error %d (%s)", e.StatusCode, e.Status)
}

func (e *HTTPError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrInternalServerError:
		return e.StatusCode == http.StatusInternalServerError
	default:
		return false
	}
}
