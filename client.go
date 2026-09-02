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

	return inv.request(http.MethodPost, "/invoices", payload, nil)
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

func (inv *Invoice) Reissue(invoiceID string) (map[string]any, error) {
	return inv.request(http.MethodPost, "/invoices/"+invoiceID+"/reissue", nil, nil)
}

type InvoiceError struct {
	Message string
}

func (e *InvoiceError) Error() string {
	return e.Message
}

func (inv *Invoice) request(method, path string, payload any, params url.Values) (map[string]any, error) {
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

	if result, ok := parsed["result"].(map[string]any); ok {
		return result, nil
	}
	return parsed, nil
}
