package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ValidationFinding is one refusal or blocker found by Validate. Field is
// the request path of the offending value, empty for a blocker of the
// company itself.
type ValidationFinding struct {
	Code           string `json:"code"`
	Field          string `json:"field"`
	Message        string `json:"message"`
	SefazRejection string `json:"sefaz_rejection"`
	Expected       string `json:"expected"`
	Received       string `json:"received"`
	FixURL         string `json:"fix_url"`
}

// ValidationWarning is a default the issuance would declare for the company.
type ValidationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

// ValidationResult is what Validate answers. Valid is false when Errors is
// not empty.
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationFinding `json:"errors"`
	Warnings []ValidationWarning `json:"warnings"`
}

// Validate dry-runs an issuance: the same request as Issue, checked by the
// API without transmitting, numbering or charging. The rules failing is a
// result with Valid false, not an error; any other refusal (malformed body,
// unknown client or item, credentials, rate limit) is returned as the
// error Issue would return.
func (inv *Invoice) Validate(req IssueRequest) (*ValidationResult, error) {
	payload, err := buildIssuePayload(req)
	if err != nil {
		return nil, err
	}

	raw, err := inv.send(http.MethodPost, "/invoices/validate", payload, nil,
		withAcceptLanguage(req.AcceptLanguage))
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnprocessableEntity {
			var result ValidationResult
			var probe map[string]json.RawMessage
			if json.Unmarshal(apiErr.body, &probe) == nil {
				if _, ok := probe["valid"]; ok && json.Unmarshal(apiErr.body, &result) == nil {
					return &result, nil
				}
			}
		}
		return nil, err
	}

	var result ValidationResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
	}
	return &result, nil
}
