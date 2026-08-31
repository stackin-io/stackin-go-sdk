package stackin

import "fmt"

type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s", e.StatusCode, e.Detail)
}

type ConnectionFailedError struct {
	Message string
}

func (e *ConnectionFailedError) Error() string {
	return e.Message
}
