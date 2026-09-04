package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackin-io/stackin-go-sdk/br"
)

// validNFEAddress is the buyer address NFE issuance requires — every field
// filled, so tests exercising something other than address validation pass it.
func validNFEAddress() *Address {
	return &Address{
		Street:       "Rua das Flores",
		Number:       "1200",
		Neighborhood: "Centro",
		City:         "Joinville",
		State:        "SC",
		ZipCode:      "89201100",
		CityCode:     "4209102",
	}
}

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

func TestIssueRequiresRecipientAddressForNFE(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com"))
	ncm := "12345678"
	cfop := "5102"

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10, NCM: &ncm, CFOP: &cfop}},
	})

	if err == nil {
		t.Fatal("expected error for missing recipient address, got nil")
	}
	if _, ok := err.(*InvoiceError); !ok {
		t.Errorf("error = %T, want *InvoiceError", err)
	}
}

func TestIssueRejectsPartialRecipientAddressForNFE(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com"))
	ncm := "12345678"
	cfop := "5102"

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFE,
		ClientName:       "Acme",
		TaxID:            "123",
		Items:            []br.Product{{Description: "Widget", Amount: 10, NCM: &ncm, CFOP: &cfop}},
		RecipientAddress: &Address{State: "SC"},
	})

	if err == nil {
		t.Fatal("expected error for partial recipient address, got nil")
	}
	if !strings.Contains(err.Error(), "CityCode") {
		t.Errorf("error = %q, want it to name the missing fields", err.Error())
	}
}

func TestIssueAllowsNFSEWithoutRecipientAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key"))

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Consulting", Amount: 10}},
	})

	if err != nil {
		t.Fatalf("nfse shouldn't need a recipient address: %v", err)
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
		DocumentType:     NFE,
		ClientName:       "Acme",
		TaxID:            "123",
		Items:            []br.Product{{Description: "Widget", Amount: 10, NCM: &ncm, CFOP: &cfop}},
		RecipientAddress: validNFEAddress(),
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

func TestIssueRequiresCFOPWhenNCMPresent(t *testing.T) {
	inv := NewInvoice(WithBaseURL("https://example.com"))
	ncm := "12345678"

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10, NCM: &ncm}},
	})

	if err == nil {
		t.Fatal("expected error for missing cfop, got nil")
	}
	if _, ok := err.(*InvoiceError); !ok {
		t.Errorf("error = %T, want *InvoiceError", err)
	}
}

func TestIssueIncludesRecipientAddressSeriesAndNumber(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "123",
		Items:            []br.Product{{Description: "Servico", Amount: 10}},
		RecipientAddress: &Address{State: "SP"},
		Series:           "1",
		Number:           "42",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["series"] != "1" {
		t.Errorf("body[series] = %v, want %q", body["series"], "1")
	}
	if body["number"] != "42" {
		t.Errorf("body[number] = %v, want %q", body["number"], "42")
	}
	if body["recipient_address"] == nil {
		t.Error("body[recipient_address] absent, want present")
	}
}

func TestWithTimeoutOption(t *testing.T) {
	inv := NewInvoice(WithTimeout(5 * time.Second))

	if inv.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", inv.Timeout)
	}
}

func TestWithAPIKeyOption(t *testing.T) {
	inv := NewInvoice(WithAPIKey("abc"))

	if inv.APIKey != "abc" {
		t.Errorf("APIKey = %q, want %q", inv.APIKey, "abc")
	}
}

func TestInvoiceErrorMessage(t *testing.T) {
	err := &InvoiceError{Message: "boom"}

	if err.Error() != "boom" {
		t.Errorf("Error() = %q, want %q", err.Error(), "boom")
	}
}

func TestRequestNonStringDetailIsJSONEncoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detail": []any{"tax_id is invalid", "cfop is invalid"},
		})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Consult("abc123", NFE)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.Detail != `["tax_id is invalid","cfop is invalid"]` {
		t.Errorf("Detail = %q, want JSON-encoded array", apiErr.Detail)
	}
}

func TestRequestReturnsFullBodyWhenNoResultKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Consult("abc123", NFE)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("result[status] = %v, want ok", result["status"])
	}
}

func TestRequestHandlesEmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Consult("abc123", NFE)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("result = %v, want empty map", result)
	}
}

func TestReissueSendsPostToReissuePath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_key": "reissued-key"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Reissue("inv-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/invoices/inv-1/reissue" {
		t.Errorf("path = %q, want /api/v1/invoices/inv-1/reissue", gotPath)
	}
	if result["access_key"] != "reissued-key" {
		t.Errorf("result[access_key] = %v, want reissued-key", result["access_key"])
	}
}

func TestReissueReturnsAPIErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "invoice not found"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Reissue("inv-missing")

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestIssueSendsIdempotencyKeyHeaderWhenSet(t *testing.T) {
	var gotKey string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"access_key": "abc"}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Issue(IssueRequest{
		DocumentType:   NFSE,
		ClientName:     "Buyer",
		TaxID:          "123",
		Items:          []br.Product{{Description: "Servico", Amount: 100}},
		IdempotencyKey: "idem-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "idem-1" {
		t.Errorf("Idempotency-Key = %q, want idem-1", gotKey)
	}
	if _, present := gotBody["idempotency_key"]; present {
		t.Error("idempotency_key leaked into the request body")
	}
}

func TestIssueOmitsIdempotencyKeyHeaderByDefault(t *testing.T) {
	var present bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, present = r.Header["Idempotency-Key"]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Buyer",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Servico", Amount: 100}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present {
		t.Error("Idempotency-Key sent when none was set")
	}
}

func TestReissueSendsIdempotencyKeyHeaderWhenSet(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_key": "reissued-key"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	_, err := inv.Reissue("inv-1", WithIdempotencyKey("idem-2"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "idem-2" {
		t.Errorf("Idempotency-Key = %q, want idem-2", gotKey)
	}
}

func TestCorrectPostsToTheCorrectionPath(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"status": "authorized"}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Correct("abc123", NFE, "Transportadora corrigida para Rapido Ltda")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/invoices/abc123/correction" {
		t.Errorf("path = %q, want /api/v1/invoices/abc123/correction", gotPath)
	}
	if gotBody["document_type"] != "nfe" {
		t.Errorf("document_type = %v, want nfe", gotBody["document_type"])
	}
	if result["status"] != "authorized" {
		t.Errorf("status = %v, want authorized", result["status"])
	}
}

func TestCorrectRejectsTextOutsideTheAllowedLength(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))

	for _, correction := range []string{"curto demais", strings.Repeat("a", 1001)} {
		_, err := inv.Correct("abc123", NFE, correction)

		var invErr *InvoiceError
		if !errors.As(err, &invErr) {
			t.Errorf("Correct(%d chars) error = %v, want *InvoiceError", len(correction), err)
		}
	}
	if called {
		t.Error("Correct reached the network on a locally invalid correction")
	}
}

func TestInvalidatePostsToTheInvalidationsPath(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "range-1", "status": "invalidated"})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Invalidate(InvalidationRequest{
		Series:      "1",
		NumberStart: 10,
		NumberEnd:   12,
		Reason:      "Numeracao reservada e nao utilizada por falha no ERP",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v1/invoices/invalidations" {
		t.Errorf("path = %q, want /api/v1/invoices/invalidations", gotPath)
	}
	if gotBody["number_start"] != float64(10) || gotBody["number_end"] != float64(12) {
		t.Errorf("range = %v..%v, want 10..12", gotBody["number_start"], gotBody["number_end"])
	}
	if result["status"] != "invalidated" {
		t.Errorf("status = %v, want invalidated", result["status"])
	}
}

func TestInvalidateRejectsBadInputBeforeTheNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	valid := "Numeracao reservada e nao utilizada por falha no ERP"

	cases := []InvalidationRequest{
		{Series: "1", NumberStart: 10, NumberEnd: 12, Reason: "curto"},
		{Series: "1", NumberStart: 10, NumberEnd: 12, Reason: strings.Repeat("a", 256)},
		{Series: "1", NumberStart: 12, NumberEnd: 10, Reason: valid},
	}
	for _, req := range cases {
		_, err := inv.Invalidate(req)

		var invErr *InvoiceError
		if !errors.As(err, &invErr) {
			t.Errorf("Invalidate(%+v) error = %v, want *InvoiceError", req, err)
		}
	}
	if called {
		t.Error("Invalidate reached the network on locally invalid input")
	}
}

func TestUnknownResponseFieldReachesTheCaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{
			"access_key":               "ABC",
			"status":                   "authorized",
			"field_invented_next_year": map[string]any{"nested": []any{1, 2}},
		}})
	}))
	defer server.Close()

	inv := NewInvoice(WithBaseURL(server.URL))
	result, err := inv.Consult("ABC", NFSE)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["access_key"] != "ABC" {
		t.Errorf("access_key = %v, want ABC", result["access_key"])
	}
	if _, present := result["field_invented_next_year"]; !present {
		t.Error("an unknown field was dropped; the API may add fields inside v1")
	}
}
