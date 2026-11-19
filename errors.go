package stackin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// PaymentRequiredError is any 402: the plan's allowance is spent and
// neither an overage price nor the account credit covers the call. Detail
// says why. errors.As
// still finds the *APIError inside.
type PaymentRequiredError struct {
	APIError
}

func (e *PaymentRequiredError) Unwrap() error {
	return &e.APIError
}

// RateLimitedError is any 429. RetryAfter is the Retry-After header in
// seconds, zero when absent. RetryAt and Code come from the body when it
// carries them: DISTRIBUTION_COOLDOWN is the SEFAZ's one-hour wait after
// a received-invoices read that found nothing more. errors.As still finds
// the *APIError inside.
type RateLimitedError struct {
	APIError
	Code       string
	RetryAfter int
	RetryAt    *time.Time
}

func (e *RateLimitedError) Unwrap() error {
	return &e.APIError
}

// ClientInactiveError is the API's 422 CLIENT_INACTIVE: the saved client
// named by ClientID is deactivated. Reactivate it, or issue without it.
// errors.As still finds the *APIError inside.
type ClientInactiveError struct {
	APIError
	Message  string
	ClientID string
}

func (e *ClientInactiveError) Unwrap() error {
	return &e.APIError
}

// ClientNotFoundError is the API's 404 CLIENT_NOT_FOUND: no saved client
// of this company has that id. errors.As still finds the *APIError inside.
type ClientNotFoundError struct {
	APIError
	Message  string
	ClientID string
}

func (e *ClientNotFoundError) Unwrap() error {
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

type rateLimitBody struct {
	Code    string `json:"code"`
	RetryAt string `json:"retry_at"`
}

func newAPIError(status int, detail string, raw json.RawMessage, body []byte, header http.Header) error {
	base := APIError{StatusCode: status, Detail: detail}

	switch status {
	case http.StatusPaymentRequired:
		return &PaymentRequiredError{APIError: base}
	case http.StatusTooManyRequests:
		return newRateLimitedError(base, raw, body, header)
	}

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
	case status == http.StatusUnprocessableEntity && parsed.Code == "CLIENT_INACTIVE":
		return &ClientInactiveError{APIError: base, Message: parsed.Message, ClientID: parsed.ClientID}
	case status == http.StatusNotFound && parsed.Code == "CLIENT_NOT_FOUND":
		return &ClientNotFoundError{APIError: base, Message: parsed.Message, ClientID: parsed.ClientID}
	}
	return &base
}

func newRateLimitedError(base APIError, raw json.RawMessage, body []byte, header http.Header) *RateLimitedError {
	limited := &RateLimitedError{APIError: base, RetryAfter: parseRetryAfter(header.Get("Retry-After"))}

	var fields rateLimitBody
	if len(body) > 0 {
		_ = json.Unmarshal(body, &fields)
	}
	if fields.Code == "" && fields.RetryAt == "" && len(raw) > 0 {
		_ = json.Unmarshal(raw, &fields)
	}

	limited.Code = fields.Code
	if at, ok := parseAPITime(fields.RetryAt); ok {
		limited.RetryAt = &at
	}
	return limited
}

func parseRetryAfter(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return seconds
	}
	if at, err := http.ParseTime(value); err == nil {
		if wait := int(time.Until(at).Seconds()); wait > 0 {
			return wait
		}
	}
	return 0
}
