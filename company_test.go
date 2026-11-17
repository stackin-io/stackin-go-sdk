package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"reflect"
	"sync"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

type recorded struct {
	mu       sync.Mutex
	requests []*http.Request
	dumps    [][]byte
}

func (r *recorded) all() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Request(nil), r.requests...)
}

func (r *recorded) bodies() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]byte(nil), r.dumps...)
}

func companyServer(t *testing.T, status int, body any) (string, *recorded) {
	t.Helper()
	rec := &recorded{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, _ := httputil.DumpRequest(r, true)
		rec.mu.Lock()
		rec.requests = append(rec.requests, r.Clone(r.Context()))
		rec.dumps = append(rec.dumps, dump)
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)

	return server.URL, rec
}

func issueNFSE(inv *Invoice) error {
	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10}},
	})
	return err
}

func companyListing() []map[string]any {
	return []map[string]any{
		{"company_id": "c-1", "name": "One", "tax_id": "00000000000191", "permissions": []string{"invoices:write"}},
		{"company_id": "c-2", "name": "Two", "tax_id": "11111111000191", "permissions": []string{"invoices:read"}},
	}
}

func TestWithoutACompanyNoHeaderIsSent(t *testing.T) {
	url, rec := companyServer(t, http.StatusOK, map[string]any{})

	if err := issueNFSE(NewInvoice(WithBaseURL(url), WithAPIKey("k"))); err != nil {
		t.Fatal(err)
	}

	if got := rec.all()[0].Header; len(got["X-Stackin-Company"]) > 0 {
		t.Errorf("X-Stackin-Company = %q, want absent", got.Get("X-Stackin-Company"))
	}
}

func TestAnEmptyCompanyLeavesTheRequestByteIdentical(t *testing.T) {
	url, rec := companyServer(t, http.StatusOK, map[string]any{})

	if err := issueNFSE(NewInvoice(WithBaseURL(url), WithAPIKey("k"))); err != nil {
		t.Fatal(err)
	}
	if err := issueNFSE(NewInvoice(WithBaseURL(url), WithAPIKey("k"), WithCompany(""))); err != nil {
		t.Fatal(err)
	}
	if err := issueNFSE(NewInvoice(WithBaseURL(url), WithAPIKey("k")).ForCompany("")); err != nil {
		t.Fatal(err)
	}

	dumps := rec.bodies()
	for i := 1; i < len(dumps); i++ {
		if string(dumps[i]) != string(dumps[0]) {
			t.Errorf("request %d differs:\n%s\nwant:\n%s", i, dumps[i], dumps[0])
		}
	}
}

func TestWithCompanySendsTheHeaderOnEveryRequest(t *testing.T) {
	url, rec := companyServer(t, http.StatusOK, map[string]any{})
	opts := []Option{WithBaseURL(url), WithAPIKey("k"), WithCompany("00.000.000/0001-91")}

	inv := NewInvoice(opts...)
	if err := issueNFSE(inv); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Consult("abc", NFSE); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Cancel("abc", NFSE, "Typo in the name"); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Companies(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFiscalReference(opts...).NCM.Get("84716052"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTaxpayer(opts...).Get("00000000000191"); err != nil {
		t.Fatal(err)
	}

	requests := rec.all()
	if len(requests) != 6 {
		t.Fatalf("%d requests, want 6", len(requests))
	}
	for _, r := range requests {
		if got := r.Header.Get("X-Stackin-Company"); got != "00.000.000/0001-91" {
			t.Errorf("%s %s: X-Stackin-Company = %q", r.Method, r.URL.Path, got)
		}
	}
}

func TestForCompanyBindsOnlyTheChainedCalls(t *testing.T) {
	url, rec := companyServer(t, http.StatusOK, map[string]any{})
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"), WithCompany("c-1"))

	if _, err := inv.ForCompany("c-2").Consult("abc", NFSE); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Consult("abc", NFSE); err != nil {
		t.Fatal(err)
	}

	requests := rec.all()
	if got := requests[0].Header.Get("X-Stackin-Company"); got != "c-2" {
		t.Errorf("chained call sent %q, want c-2", got)
	}
	if got := requests[1].Header.Get("X-Stackin-Company"); got != "c-1" {
		t.Errorf("client sent %q after ForCompany, want c-1", got)
	}
}

func TestCompaniesReadsTheAuthorizedCompanies(t *testing.T) {
	url, rec := companyServer(t, http.StatusOK, map[string]any{
		"environment": "production",
		"companies":   companyListing(),
	})

	got, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Companies()
	if err != nil {
		t.Fatal(err)
	}

	r := rec.all()[0]
	if r.Method != http.MethodGet || r.URL.Path != "/api/v1/me/companies" {
		t.Errorf("request = %s %s, want GET /api/v1/me/companies", r.Method, r.URL.Path)
	}
	want := &AuthorizedCompanies{
		Environment: "production",
		Companies: []AuthorizedCompany{
			{CompanyID: "c-1", Name: "One", TaxID: "00000000000191", Permissions: []string{"invoices:write"}},
			{CompanyID: "c-2", Name: "Two", TaxID: "11111111000191", Permissions: []string{"invoices:read"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Companies() = %+v, want %+v", got, want)
	}
}

func TestCompaniesRefusesAnUnexpectedShape(t *testing.T) {
	url, _ := companyServer(t, http.StatusOK, []string{"not", "an", "object"})

	_, err := NewInvoice(WithBaseURL(url)).Companies()
	if _, ok := err.(*InvoiceError); !ok {
		t.Fatalf("error = %T, want *InvoiceError", err)
	}
}

func TestCompanyRequiredIsTypedAndNotRetried(t *testing.T) {
	url, rec := companyServer(t, http.StatusUnprocessableEntity, map[string]any{
		"detail": map[string]any{
			"error":             "company_required",
			"error_description": "This credential covers several companies.",
			"companies":         companyListing(),
		},
	})

	_, err := NewInvoice(WithBaseURL(url)).Consult("abc", NFSE)

	var typed *CompanyRequiredError
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *CompanyRequiredError", err)
	}
	if len(typed.Companies) != 2 || typed.Companies[1].CompanyID != "c-2" {
		t.Errorf("Companies = %+v", typed.Companies)
	}
	if typed.Description != "This credential covers several companies." {
		t.Errorf("Description = %q", typed.Description)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("errors.As(*APIError) = %v", apiErr)
	}
	if n := len(rec.all()); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestCompanyNotInScopeIsTypedAndNotRetried(t *testing.T) {
	url, rec := companyServer(t, http.StatusForbidden, map[string]any{
		"detail": map[string]any{
			"error":             "company_not_in_scope",
			"error_description": "This credential does not cover company 22222222000191.",
			"companies":         companyListing(),
		},
	})

	_, err := NewInvoice(WithBaseURL(url), WithCompany("22222222000191")).Consult("abc", NFSE)

	var typed *CompanyNotInScopeError
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *CompanyNotInScopeError", err)
	}
	if len(typed.Companies) != 2 || typed.Companies[0].TaxID != "00000000000191" {
		t.Errorf("Companies = %+v", typed.Companies)
	}
	if typed.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", typed.StatusCode)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As(*APIError) failed")
	}
	if n := len(rec.all()); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestInsufficientPermissionIsTypedAndNotRetried(t *testing.T) {
	description := "INSUFFICIENT_PERMISSION: needs invoices:write, has invoices:read, company=c-2"
	url, rec := companyServer(t, http.StatusForbidden, map[string]any{
		"detail": map[string]any{
			"error":             "insufficient_scope",
			"error_description": description,
			"code":              "INSUFFICIENT_PERMISSION",
		},
	})

	err := issueNFSE(NewInvoice(WithBaseURL(url), WithCompany("c-2")))

	var typed *InsufficientPermissionError
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *InsufficientPermissionError", err)
	}
	if typed.Description != description {
		t.Errorf("Description = %q", typed.Description)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("errors.As(*APIError) = %v", apiErr)
	}
	if n := len(rec.all()); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestAnOtherStructuredErrorStaysAnAPIError(t *testing.T) {
	url, _ := companyServer(t, http.StatusForbidden, map[string]any{
		"detail": map[string]any{"error": "something_else"},
	})

	_, err := NewInvoice(WithBaseURL(url)).Consult("abc", NFSE)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.Detail != `{"error":"something_else"}` {
		t.Errorf("Detail = %q", apiErr.Detail)
	}
}

func TestATypedErrorKeepsTheAPIErrorMessage(t *testing.T) {
	url, _ := companyServer(t, http.StatusUnprocessableEntity, map[string]any{
		"detail": map[string]any{"error": "company_required", "companies": []any{}},
	})

	_, err := NewInvoice(WithBaseURL(url)).Consult("abc", NFSE)

	if got, want := err.Error(), `[422] {"companies":[],"error":"company_required"}`; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
