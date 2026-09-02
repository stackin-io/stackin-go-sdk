package stackin

import "testing"

func TestAPIErrorMessage(t *testing.T) {
	err := &APIError{StatusCode: 404, Detail: "not found"}

	if got, want := err.Error(), "[404] not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestConnectionFailedErrorMessage(t *testing.T) {
	err := &ConnectionFailedError{Message: "could not connect"}

	if got, want := err.Error(), "could not connect"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
