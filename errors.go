package stackin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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

// CompanyRequiredError is the API's 422 company_required: the credential
// covers several companies and the request named none. Companies lists
// them; name one with WithCompany or ForCompany. Retrying the same request
// cannot succeed. errors.As still finds the *APIError inside.
type CompanyRequiredError struct {
	APIError
	Description string
	Companies   []AuthorizedCompany
}

func (e *CompanyRequiredError) Unwrap() error {
	return &e.APIError
}

// CompanyNotInScopeError is the API's 403 company_not_in_scope: the
// company named is not one this credential covers. Companies lists the
// ones it does. Retrying the same request cannot succeed. errors.As still
// finds the *APIError inside.
type CompanyNotInScopeError struct {
	APIError
	Description string
	Companies   []AuthorizedCompany
}

func (e *CompanyNotInScopeError) Unwrap() error {
	return &e.APIError
}

// InsufficientPermissionError is the API's 403 INSUFFICIENT_PERMISSION:
// the credential reaches the company but lacks the permission the call
// needs. Description reads "needs X, has Y, company=<id>". Retrying the
// same request cannot succeed. errors.As still finds the *APIError inside.
type InsufficientPermissionError struct {
	APIError
	Description string
}

func (e *InsufficientPermissionError) Unwrap() error {
	return &e.APIError
}

// ClientAlreadyExistsError is the API's 409 CLIENT_EXISTS: the company
// already has a client with this tax id or foreign id. ClientID names it,
// to read or reactivate instead of searching for it. errors.As still
// finds the *APIError inside.
type ClientAlreadyExistsError struct {
	APIError
	Message  string
	ClientID string
}

func (e *ClientAlreadyExistsError) Unwrap() error {
	return &e.APIError
}

type companyErrorDetail struct {
	Error       string              `json:"error"`
	Code        string              `json:"code"`
	Description string              `json:"error_description"`
	Companies   []AuthorizedCompany `json:"companies"`
	Message     string              `json:"message"`
	ClientID    string              `json:"client_id"`
}

func newAPIError(status int, detail string, raw json.RawMessage) error {
	base := APIError{StatusCode: status, Detail: detail}

	var parsed companyErrorDetail
	if len(raw) == 0 || json.Unmarshal(raw, &parsed) != nil {
		return &base
	}

	switch {
	case status == http.StatusUnprocessableEntity && parsed.Error == "company_required":
		return &CompanyRequiredError{APIError: base, Description: parsed.Description, Companies: parsed.Companies}
	case status == http.StatusForbidden && parsed.Error == "company_not_in_scope":
		return &CompanyNotInScopeError{APIError: base, Description: parsed.Description, Companies: parsed.Companies}
	case status == http.StatusForbidden && parsed.Code == "INSUFFICIENT_PERMISSION":
		return &InsufficientPermissionError{APIError: base, Description: parsed.Description}
	case status == http.StatusConflict && parsed.Code == "CLIENT_EXISTS":
		return &ClientAlreadyExistsError{APIError: base, Message: parsed.Message, ClientID: parsed.ClientID}
	}
	return &base
}
