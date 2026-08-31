package stackin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

func TestNewInvoiceDefaultsToSDKHost(t *testing.T) {
	t.Setenv("STACKIN_BASE_URL", "")
	t.Setenv("STACKIN_ENVIRONMENT", "")
	t.Setenv("STACKIN_API_KEY", "")

	inv := NewInvoice()

	if inv.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", inv.BaseURL, DefaultBaseURL)
	}
}

func TestNewInvoiceUsesExplicitBaseURL(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com/"))

	if inv.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", inv.BaseURL)
	}
}

func TestNewInvoiceUsesEnvironmentOption(t *testing.T) {
	inv := NewInvoice(WithEnvironment(Local))

	if inv.BaseURL != "http://localhost:8000" {
		t.Errorf("BaseURL = %q, want local environment URL", inv.BaseURL)
	}
}

func TestResolveBaseURLPrefersExplicitURLOverEnvironment(t *testing.T) {
	t.Setenv("STACKIN_BASE_URL", "https://override.example.com")
	t.Setenv("STACKIN_ENVIRONMENT", "local")

	if got := resolveBaseURL(); got != "https://override.example.com" {
		t.Errorf("resolveBaseURL() = %q, want explicit STACKIN_BASE_URL to win", got)
	}
}

func TestResolveBaseURLFallsBackToEnvironment(t *testing.T) {
	t.Setenv("STACKIN_BASE_URL", "")
	t.Setenv("STACKIN_ENVIRONMENT", "local")

	if got := resolveBaseURL(); got != "http://localhost:8000" {
		t.Errorf("resolveBaseURL() = %q, want local environment URL", got)
	}
}

func TestIssueRejectsEmptyItems(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com"))

	_, err := inv.Issue(IssueRequest{DocumentType: NFSE, ClientName: "Acme", TaxID: "123"})

	if err == nil {
		t.Fatal("expected error for empty items, got nil")
	}
	if _, ok := err.(*InvoiceError); !ok {
		t.Errorf("error = %T, want *InvoiceError", err)
	}
}

func TestIssueRequiresNCMAndCFOPForNFE(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com"))

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10}},
	})

	if err == nil {
		t.Fatal("expected error for missing ncm/cfop, got nil")
	}
	if _, ok := err.(*InvoiceError); !ok {
		t.Errorf("error = %T, want *InvoiceError", err)
	}
}

func TestIssuePostsToInvoicesEndpointWithAuthHeader(t *testing.T) {
	var gotAuth, gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"access_key": "abc123"},
		})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key"))
	ncm := "12345678"
	cfop := "5102"

	result, err := inv.Issue(IssueRequest{
		DocumentType: NFE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10, NCM: &ncm, CFOP: &cfop}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/invoices" {
		t.Errorf("path = %q, want /api/v1/invoices", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization header = %q, want Bearer test-key", gotAuth)
	}
	if result["access_key"] != "abc123" {
		t.Errorf("result[access_key] = %v, want abc123", result["access_key"])
	}
}

func TestRequestOmitsAuthHeaderWithoutAPIKey(t *testing.T) {
	sawHeader := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHeader = len(r.Header["Authorization"]) > 0
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL), WithAPIKey(""))
	_, err := inv.Consult("abc123", NFSE)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawHeader {
		t.Error("Authorization header present, want absent without an API key")
	}
}

func TestRequestReturnsAPIErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "tax_id is invalid"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Consult("abc123", NFE)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
	if apiErr.Detail != "tax_id is invalid" {
		t.Errorf("Detail = %q, want %q", apiErr.Detail, "tax_id is invalid")
	}
}

func TestRequestReturnsConnectionFailedErrorOnUnreachableHost(t *testing.T) {
	inv := NewInvoice(WithBaseURL("http://127.0.0.1:1"))

	_, err := inv.Consult("abc123", NFE)

	if _, ok := err.(*ConnectionFailedError); !ok {
		t.Errorf("error = %T, want *ConnectionFailedError", err)
	}
}

func TestCancelSendsReasonAndDocumentType(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"status": "cancelled"}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Cancel("abc123", NFSE, "duplicate issuance")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["reason"] != "duplicate issuance" {
		t.Errorf("body[reason] = %v, want %q", body["reason"], "duplicate issuance")
	}
	if body["document_type"] != string(NFSE) {
		t.Errorf("body[document_type] = %v, want %q", body["document_type"], NFSE)
	}
}
