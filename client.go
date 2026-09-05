// Package stackin is the official Go SDK for the stackin fiscal
// document API — issue, consult, and cancel Brazilian NF-e and NFS-e
// electronic invoices with a single API key, no certificates or XML
// handling required.
//
// See https://docs.stackin.io for the REST API reference and
// https://docs.stackin.io/blog/go-sdk for the Go SDK guide.
package stackin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stackin-io/stackin-go-sdk/br"
)

const DefaultBaseURL = "https://sdk.stackin.io"

var environmentURLs = map[Environment]string{
	Local:      "http://localhost:8000",
	Test:       DefaultBaseURL,
	Production: DefaultBaseURL,
}

type Invoice struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	httpClient *http.Client
}

type Option func(*Invoice)

func WithBaseURL(baseURL string) Option {
	return func(i *Invoice) { i.BaseURL = baseURL }
}

func WithEnvironment(env Environment) Option {
	return func(i *Invoice) { i.BaseURL = environmentURLs[env] }
}

func WithAPIKey(apiKey string) Option {
	return func(i *Invoice) { i.APIKey = apiKey }
}

func WithTimeout(timeout time.Duration) Option {
	return func(i *Invoice) { i.Timeout = timeout }
}

func NewInvoice(opts ...Option) *Invoice {
	i := &Invoice{
		BaseURL: resolveBaseURL(),
		APIKey:  os.Getenv("STACKIN_API_KEY"),
		Timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(i)
	}
	i.BaseURL = strings.TrimRight(i.BaseURL, "/")
	i.httpClient = &http.Client{Timeout: i.Timeout}
	return i
}

func resolveBaseURL() string {
	if v := os.Getenv("STACKIN_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("STACKIN_ENVIRONMENT"); v != "" {
		if u, ok := environmentURLs[Environment(v)]; ok {
			return u
		}
	}
	return DefaultBaseURL
}

type IssueRequest struct {
	DocumentType     DocumentType
	ClientName       string
	TaxID            string
	Items            []br.Product
	RecipientAddress *Address
	Series           string
	Number           string

	// IdempotencyKey makes a retry safe: the same key with the same body
	// replays the first response instead of issuing a second document.
	// Keys live 24 hours. Never generated for you — only the caller knows
	// which two requests are meant to be the same one.
	IdempotencyKey string
}

// RequestOption tunes a single call, as opposed to Option, which tunes the
// client.
type RequestOption func(*requestConfig)

type requestConfig struct {
	idempotencyKey string
}

// WithIdempotencyKey attaches an Idempotency-Key to one call. See
// IssueRequest.IdempotencyKey for what the API does with it.
func WithIdempotencyKey(key string) RequestOption {
	return func(cfg *requestConfig) { cfg.idempotencyKey = key }
}

// validateNFEAddress rejects a missing or partial buyer address before the
// network call — the SEFAZ rejects a partial enderDest (274/726/696/695).
func validateNFEAddress(address *Address) error {
	if address == nil {
		return &InvoiceError{Message: "RecipientAddress is required for NFE"}
	}

	fields := []struct {
		name  string
		value string
	}{
		{"Street", address.Street},
		{"Number", address.Number},
		{"Neighborhood", address.Neighborhood},
		{"City", address.City},
		{"State", address.State},
		{"ZipCode", address.ZipCode},
		{"CityCode", address.CityCode},
	}

	var missing []string
	for _, field := range fields {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return &InvoiceError{Message: fmt.Sprintf(
			"RecipientAddress is missing required fields for NFE: %s",
			strings.Join(missing, ", "),
		)}
	}

	return nil
}

func (inv *Invoice) Issue(req IssueRequest) (map[string]any, error) {
	if len(req.Items) == 0 {
		return nil, &InvoiceError{Message: "items can't be empty"}
	}

	if req.DocumentType == NFE {
		for i, item := range req.Items {
			if item.NCM == nil || *item.NCM == "" {
				return nil, &InvoiceError{Message: fmt.Sprintf("items[%d].ncm is required for NFE", i)}
			}
			if item.CFOP == nil || *item.CFOP == "" {
				return nil, &InvoiceError{Message: fmt.Sprintf("items[%d].cfop is required for NFE", i)}
			}
		}
		if err := validateNFEAddress(req.RecipientAddress); err != nil {
			return nil, err
		}
	}

	items := make([]map[string]any, len(req.Items))
	for i, item := range req.Items {
		items[i] = item.ToDict()
	}

	payload := map[string]any{
		"document_type": string(req.DocumentType),
		"client_name":   req.ClientName,
		"tax_id":        req.TaxID,
		"items":         items,
	}
	if req.RecipientAddress != nil {
		payload["recipient_address"] = req.RecipientAddress
	}
	if req.Series != "" {
		payload["series"] = req.Series
	}
	if req.Number != "" {
		payload["number"] = req.Number
	}

	return inv.request(http.MethodPost, "/invoices", payload, nil,
		WithIdempotencyKey(req.IdempotencyKey))
}

func (inv *Invoice) Consult(accessKey string, documentType DocumentType) (map[string]any, error) {
	params := url.Values{"document_type": {string(documentType)}}
	return inv.request(http.MethodGet, "/invoices/"+accessKey, nil, params)
}

func (inv *Invoice) Cancel(accessKey string, documentType DocumentType, reason string) (map[string]any, error) {
	payload := map[string]any{
		"document_type": string(documentType),
		"reason":        reason,
	}
	return inv.request(http.MethodPost, "/invoices/"+accessKey+"/cancel", payload, nil)
}

type InvalidationRequest struct {
	Series      string
	NumberStart int
	NumberEnd   int
	Reason      string
}

func (inv *Invoice) Invalidate(req InvalidationRequest) (map[string]any, error) {
	length := len([]rune(req.Reason))
	if length < 15 || length > 255 {
		return nil, &InvoiceError{Message: "reason must be 15 to 255 characters"}
	}
	if req.NumberEnd < req.NumberStart {
		return nil, &InvoiceError{Message: "NumberEnd can't be below NumberStart"}
	}

	payload := map[string]any{
		"series":       req.Series,
		"number_start": req.NumberStart,
		"number_end":   req.NumberEnd,
		"reason":       req.Reason,
	}
	return inv.request(http.MethodPost, "/invoices/invalidations", payload, nil)
}

func (inv *Invoice) Correct(accessKey string, documentType DocumentType, correction string) (map[string]any, error) {
	length := len([]rune(correction))
	if length < 15 || length > 1000 {
		return nil, &InvoiceError{Message: "correction must be 15 to 1000 characters"}
	}

	payload := map[string]any{
		"document_type": string(documentType),
		"correction":    correction,
	}
	return inv.request(http.MethodPost, "/invoices/"+accessKey+"/correction", payload, nil)
}

func (inv *Invoice) Reissue(invoiceID string, opts ...RequestOption) (map[string]any, error) {
	return inv.request(http.MethodPost, "/invoices/"+invoiceID+"/reissue", nil, nil, opts...)
}

type InvoiceError struct {
	Message string
}

func (e *InvoiceError) Error() string {
	return e.Message
}

// Received lists documents other companies issued against this one.
//
// Reads what the API already collected; it does not call the authorizer.
// Collecting runs on a schedule there, because the SEFAZ caps how many
// times a CNPJ may ask per day.
func (inv *Invoice) Received(limit, offset int) (map[string]any, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	return inv.request(http.MethodGet, "/received-invoices", nil, params)
}

// Manifest files the recipient's formal answer to a received document.
//
// Only OperacaoNaoRealizada takes a reason, and it requires one. Both are
// fixed rules, checked here rather than spending a round trip to be told.
func (inv *Invoice) Manifest(accessKey string, manifestation Manifestation, reason string) (map[string]any, error) {
	needsReason := manifestation == OperacaoNaoRealizada
	if needsReason && reason == "" {
		return nil, &InvoiceError{
			Message: "manifestation 210240 (operacao nao realizada) requires a reason",
		}
	}
	if !needsReason && reason != "" {
		return nil, &InvoiceError{
			Message: fmt.Sprintf("manifestation %s does not take a reason", manifestation),
		}
	}

	payload := map[string]any{"manifestation": string(manifestation)}
	if reason != "" {
		payload["reason"] = reason
	}
	return inv.request(
		http.MethodPost,
		"/received-invoices/"+accessKey+"/manifestation",
		payload,
		nil,
	)
}

// Pdf returns the authorizer's own rendering of an authorized document,
// as raw bytes — the only method that does not return a parsed object.
// The XML is the legally valid document; this is a convenience, and the
// authorizer's endpoint for it is unstable, so an APIError with status
// 502 means the authorizer is unavailable, not that the invoice is
// wrong. NFS-e only.
func (inv *Invoice) Pdf(accessKey string, documentType DocumentType) ([]byte, error) {
	params := url.Values{"document_type": {string(documentType)}}
	return inv.send(http.MethodGet, "/invoices/"+accessKey+"/pdf", nil, params)
}

func (inv *Invoice) request(method, path string, payload any, params url.Values, opts ...RequestOption) (map[string]any, error) {
	raw, err := inv.send(method, path, payload, params, opts...)
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	if result, ok := parsed["result"].(map[string]any); ok {
		return result, nil
	}
	return parsed, nil
}

func (inv *Invoice) send(method, path string, payload any, params url.Values, opts ...RequestOption) ([]byte, error) {
	var cfg requestConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	reqURL := inv.BaseURL + "/api/v1" + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}

	httpReq, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, &ConnectionFailedError{Message: err.Error()}
	}
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if inv.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+inv.APIKey)
	}
	if cfg.idempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", cfg.idempotencyKey)
	}

	resp, err := inv.httpClient.Do(httpReq)
	if err != nil {
		return nil, &ConnectionFailedError{Message: err.Error()}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ConnectionFailedError{Message: err.Error()}
	}

	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := string(raw)
		if d, ok := parsed["detail"]; ok {
			if s, ok := d.(string); ok {
				detail = s
			} else if b, err := json.Marshal(d); err == nil {
				detail = string(b)
			}
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Detail: detail}
	}

	return raw, nil
}
