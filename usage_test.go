package stackin

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestUsageReadsEveryField(t *testing.T) {
	var seen []*http.Request
	url := jsonServer(t, http.StatusOK, nil, `{
		"plan_name": "Pro", "plan_slug": "pro",
		"included_invoices": 500, "used_invoices": 42, "overage_price_cents": 15,
		"included_received": 100, "used_received": 7, "received_price_cents": 3, "received_free": true,
		"current_period_start": "2026-11-01T00:00:00Z", "current_period_end": "2026-12-01T00:00:00Z",
		"credit_balance": 1250, "companies_on_account": 2,
		"field_invented_next_year": "ignored"
	}`, &seen)

	got, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Usage()
	if err != nil {
		t.Fatal(err)
	}

	if r := seen[0]; r.Method != http.MethodGet || r.URL.Path != "/api/v1/company/usage" {
		t.Errorf("request = %s %s, want GET /api/v1/company/usage", r.Method, r.URL.Path)
	}
	if got.PlanName != "Pro" || got.PlanSlug != "pro" {
		t.Errorf("plan = %q/%q", got.PlanName, got.PlanSlug)
	}
	if got.IncludedInvoices != 500 || got.UsedInvoices != 42 {
		t.Errorf("invoices = %d/%d", got.UsedInvoices, got.IncludedInvoices)
	}
	if got.OveragePriceCents == nil || *got.OveragePriceCents != 15 {
		t.Errorf("OveragePriceCents = %v, want 15", got.OveragePriceCents)
	}
	if got.IncludedReceived != 100 || got.UsedReceived != 7 || got.ReceivedPriceCents != 3 || !got.ReceivedFree {
		t.Errorf("received = %+v", got)
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !got.CurrentPeriodStart.Equal(want) {
		t.Errorf("CurrentPeriodStart = %v, want %v", got.CurrentPeriodStart, want)
	}
	if want := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC); got.CurrentPeriodEnd == nil || !got.CurrentPeriodEnd.Equal(want) {
		t.Errorf("CurrentPeriodEnd = %v, want %v", got.CurrentPeriodEnd, want)
	}
	if got.CreditBalance != 1250 || got.CompaniesOnAccount != 2 {
		t.Errorf("CreditBalance = %d, CompaniesOnAccount = %d", got.CreditBalance, got.CompaniesOnAccount)
	}
}

func TestUsageDefaultsTheReceivedFieldsWhenAbsent(t *testing.T) {
	url := jsonServer(t, http.StatusOK, nil, `{
		"plan_name": "Free", "plan_slug": "free",
		"included_invoices": 10, "used_invoices": 0, "overage_price_cents": null,
		"current_period_start": "2026-11-01T00:00:00", "current_period_end": null,
		"credit_balance": 0, "companies_on_account": 1
	}`, nil)

	got, err := NewInvoice(WithBaseURL(url)).Usage()
	if err != nil {
		t.Fatal(err)
	}

	if got.IncludedReceived != 50 || got.UsedReceived != 0 || got.ReceivedPriceCents != 5 || got.ReceivedFree {
		t.Errorf("received defaults = %d/%d/%d/%v, want 50/0/5/false",
			got.IncludedReceived, got.UsedReceived, got.ReceivedPriceCents, got.ReceivedFree)
	}
	if got.OveragePriceCents != nil {
		t.Errorf("OveragePriceCents = %v, want nil", *got.OveragePriceCents)
	}
	if got.CurrentPeriodEnd != nil {
		t.Errorf("CurrentPeriodEnd = %v, want nil", got.CurrentPeriodEnd)
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !got.CurrentPeriodStart.Equal(want) {
		t.Errorf("CurrentPeriodStart = %v, want %v", got.CurrentPeriodStart, want)
	}
}

func TestUsageRefusesAnUnexpectedShape(t *testing.T) {
	url := jsonServer(t, http.StatusOK, nil, `["not", "an", "object"]`, nil)

	_, err := NewInvoice(WithBaseURL(url)).Usage()
	if _, ok := err.(*InvoiceError); !ok {
		t.Fatalf("error = %T, want *InvoiceError", err)
	}
}

func TestSyncReceivedPostsWithoutABody(t *testing.T) {
	var seen []*http.Request
	url := jsonServer(t, http.StatusOK, nil, `{
		"stored": 3, "pages": 1, "last_nsu": "000000000000123", "max_nsu": "000000000000123",
		"next_sync_at": "2026-11-18T22:00:00Z"
	}`, &seen)

	got, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).SyncReceived()
	if err != nil {
		t.Fatal(err)
	}

	r := seen[0]
	if r.Method != http.MethodPost || r.URL.Path != "/api/v1/received-invoices/sync" {
		t.Errorf("request = %s %s, want POST /api/v1/received-invoices/sync", r.Method, r.URL.Path)
	}
	if r.ContentLength > 0 {
		t.Errorf("ContentLength = %d, want no body", r.ContentLength)
	}
	if got.Stored != 3 || got.Pages != 1 || got.LastNSU != "000000000000123" {
		t.Errorf("SyncReceived() = %+v", got)
	}
	if got.MaxNSU == nil || *got.MaxNSU != "000000000000123" {
		t.Errorf("MaxNSU = %v", got.MaxNSU)
	}
	if want := time.Date(2026, 11, 18, 22, 0, 0, 0, time.UTC); got.NextSyncAt == nil || !got.NextSyncAt.Equal(want) {
		t.Errorf("NextSyncAt = %v, want %v", got.NextSyncAt, want)
	}
}

func TestSyncReceivedLeavesTheOptionalFieldsNil(t *testing.T) {
	url := jsonServer(t, http.StatusOK, nil, `{"stored": 0, "pages": 0, "last_nsu": "0", "max_nsu": null}`, nil)

	got, err := NewInvoice(WithBaseURL(url)).SyncReceived()
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxNSU != nil || got.NextSyncAt != nil {
		t.Errorf("SyncReceived() = %+v, want nil MaxNSU and NextSyncAt", got)
	}
}

func TestSyncReceivedCooldownIsRateLimited(t *testing.T) {
	url := jsonServer(t, http.StatusTooManyRequests, http.Header{"Retry-After": {"1800"}}, `{
		"detail": "The SEFAZ asks to wait before the next read.",
		"code": "DISTRIBUTION_COOLDOWN",
		"retry_at": "2026-11-18T22:00:00Z"
	}`, nil)

	_, err := NewInvoice(WithBaseURL(url)).SyncReceived()

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("error = %T, want *RateLimitedError", err)
	}
	if limited.Code != "DISTRIBUTION_COOLDOWN" {
		t.Errorf("Code = %q", limited.Code)
	}
	if limited.RetryAfter != 1800 {
		t.Errorf("RetryAfter = %d, want 1800", limited.RetryAfter)
	}
	if want := time.Date(2026, 11, 18, 22, 0, 0, 0, time.UTC); limited.RetryAt == nil || !limited.RetryAt.Equal(want) {
		t.Errorf("RetryAt = %v, want %v", limited.RetryAt, want)
	}
	if limited.Detail != "The SEFAZ asks to wait before the next read." {
		t.Errorf("Detail = %q", limited.Detail)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("errors.As(*APIError) = %v, want status 429", apiErr)
	}
}
