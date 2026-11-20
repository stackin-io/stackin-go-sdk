package stackin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PageQuery is the API's pagination for a listing. Every field is
// optional; a zero value is left out and the API's own default applies.
type PageQuery struct {
	Limit   int
	Offset  int
	SortBy  string
	OrderBy string
}

func (q PageQuery) values() url.Values {
	params := url.Values{}
	if q.Limit > 0 {
		params.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		params.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.SortBy != "" {
		params.Set("sort_by", q.SortBy)
	}
	if q.OrderBy != "" {
		params.Set("order_by", q.OrderBy)
	}
	return params
}

// Invalidations lists the numbering ranges this company already
// invalidated, a page. Each row carries the range, the reason, the SEFAZ
// status and its protocol. Read it before Invalidate to avoid reporting
// the same range twice.
func (inv *Invoice) Invalidations(query PageQuery) (map[string]any, error) {
	return inv.request(http.MethodGet, "/invoices/invalidations", nil, query.values())
}

// Rejection explains why a rejected document was refused, and what to
// change. Submissions has the authorizer's own answer; this reads it for
// someone who does not speak SEFAZ: code, message, explanation, steps, the
// field to fix and, when there is one, a suggested value. It takes the
// invoiceID, like Submissions. A document that is not rejected is an
// *APIError 409.
func (inv *Invoice) Rejection(invoiceID string) (map[string]any, error) {
	id, err := segment(invoiceID)
	if err != nil {
		return nil, err
	}
	return inv.request(http.MethodGet, "/invoices/"+id+"/rejection", nil, nil)
}

// ClassifyRequest is one item's description and the context that changes
// its codes.
type ClassifyRequest struct {
	// Description is the item as the shop names it, 3 to 500 characters.
	Description string
	// DocumentType is NFE or NFSE. Required.
	DocumentType DocumentType
	// RecipientState is the buyer's UF, two letters. Omitted when empty.
	RecipientState string
	// Answers settle what the last suggestion listed in
	// missing_information: operation_scope, purpose, contains_alcohol,
	// service_city_code, tax_retained. Omitted when empty.
	Answers map[string]any
}

// Classify suggests a fiscal classification for one item's description.
//
// A suggestion, never a decision: primary_candidate and
// alternative_candidates carry the NCM, CFOP, CST/CSOSN, CEST or service
// code, each read from the published tables, with a confidence, and
// missing_information lists what would settle it (pass those back in
// Answers). Someone confirms it before it goes on an item; the codes
// decide the tax owed.
func (inv *Invoice) Classify(req ClassifyRequest) (map[string]any, error) {
	length := len([]rune(req.Description))
	if length < 3 || length > 500 {
		return nil, &InvoiceError{Message: "description must be 3 to 500 characters"}
	}
	if req.DocumentType != NFE && req.DocumentType != NFSE {
		return nil, &InvoiceError{Message: "document_type must be nfe or nfse"}
	}
	if req.RecipientState != "" && len([]rune(req.RecipientState)) != 2 {
		return nil, &InvoiceError{Message: "recipient_state must be a two-letter UF"}
	}

	payload := map[string]any{
		"description":   req.Description,
		"document_type": string(req.DocumentType),
	}
	if req.RecipientState != "" {
		payload["recipient_state"] = strings.ToUpper(req.RecipientState)
	}
	if len(req.Answers) > 0 {
		payload["answers"] = req.Answers
	}
	return inv.request(http.MethodPost, "/ai/classify", payload, nil)
}

// Stats counts this company's documents, and the last days of them.
//
// total, by_document_type and by_status cover every document;
// current_period/previous_period compare the last days (1 to 365) with the
// ones before, and daily_series has one row per day. Zero days sends
// nothing and the API answers for 30.
func (inv *Invoice) Stats(days int) (map[string]any, error) {
	params := url.Values{}
	if days != 0 {
		if days < 1 || days > 365 {
			return nil, &InvoiceError{Message: "days must be 1 to 365"}
		}
		params.Set("days", strconv.Itoa(days))
	}
	return inv.request(http.MethodGet, "/invoices/stats", nil, params)
}

// Readiness says whether the company can issue right now, and if not, why
// (GET /company).
//
// ready is false with missing naming the cause: certificate,
// certificate_expired, certificate_unreadable or address. Also
// certificate_expires_at, tax_regime (CRT), state and the default_series
// per document type. Ask before a first document instead of failing it.
func (inv *Invoice) Readiness() (map[string]any, error) {
	return inv.request(http.MethodGet, "/company", nil, nil)
}
