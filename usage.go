package stackin

import (
	"encoding/json"
	"net/http"
	"time"
)

// Usage is the plan behind the company and how much of it the current
// period has used. The numbers are the paying account's, shared by every
// company it holds; CompaniesOnAccount says whether that sharing is real.
// Prices and CreditBalance are in cents.
//
// Received invoices: the first IncludedReceived a month come with every
// plan, then each costs ReceivedPriceCents from the account credit.
// ReceivedFree is true for the accounts exempt from that charge.
type Usage struct {
	PlanName           string
	PlanSlug           string
	IncludedInvoices   int
	UsedInvoices       int
	OveragePriceCents  *int
	IncludedReceived   int
	UsedReceived       int
	ReceivedPriceCents int
	ReceivedFree       bool
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   *time.Time
	CreditBalance      int
	CompaniesOnAccount int
}

func (u *Usage) UnmarshalJSON(raw []byte) error {
	wire := struct {
		PlanName           string  `json:"plan_name"`
		PlanSlug           string  `json:"plan_slug"`
		IncludedInvoices   int     `json:"included_invoices"`
		UsedInvoices       int     `json:"used_invoices"`
		OveragePriceCents  *int    `json:"overage_price_cents"`
		IncludedReceived   *int    `json:"included_received"`
		UsedReceived       *int    `json:"used_received"`
		ReceivedPriceCents *int    `json:"received_price_cents"`
		ReceivedFree       *bool   `json:"received_free"`
		CurrentPeriodStart apiTime `json:"current_period_start"`
		CurrentPeriodEnd   apiTime `json:"current_period_end"`
		CreditBalance      int     `json:"credit_balance"`
		CompaniesOnAccount int     `json:"companies_on_account"`
	}{}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}

	*u = Usage{
		PlanName:           wire.PlanName,
		PlanSlug:           wire.PlanSlug,
		IncludedInvoices:   wire.IncludedInvoices,
		UsedInvoices:       wire.UsedInvoices,
		OveragePriceCents:  wire.OveragePriceCents,
		IncludedReceived:   intOr(wire.IncludedReceived, 50),
		UsedReceived:       intOr(wire.UsedReceived, 0),
		ReceivedPriceCents: intOr(wire.ReceivedPriceCents, 5),
		ReceivedFree:       wire.ReceivedFree != nil && *wire.ReceivedFree,
		CurrentPeriodEnd:   wire.CurrentPeriodEnd.value,
		CreditBalance:      wire.CreditBalance,
		CompaniesOnAccount: wire.CompaniesOnAccount,
	}
	if wire.CurrentPeriodStart.value != nil {
		u.CurrentPeriodStart = *wire.CurrentPeriodStart.value
	}
	return nil
}

func intOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

// SyncResult is what one read of the SEFAZ distribution brought back.
// NextSyncAt is set when the SEFAZ had nothing more to deliver: no read is
// allowed before it.
type SyncResult struct {
	Stored     int
	Pages      int
	LastNSU    string
	MaxNSU     *string
	NextSyncAt *time.Time
}

func (s *SyncResult) UnmarshalJSON(raw []byte) error {
	wire := struct {
		Stored     int     `json:"stored"`
		Pages      int     `json:"pages"`
		LastNSU    string  `json:"last_nsu"`
		MaxNSU     *string `json:"max_nsu"`
		NextSyncAt apiTime `json:"next_sync_at"`
	}{}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}

	*s = SyncResult{
		Stored:     wire.Stored,
		Pages:      wire.Pages,
		LastNSU:    wire.LastNSU,
		MaxNSU:     wire.MaxNSU,
		NextSyncAt: wire.NextSyncAt.value,
	}
	return nil
}

// Usage reads the plan behind this company and how much of it the
// current period has used (GET /company/usage).
func (inv *Invoice) Usage() (*Usage, error) {
	var usage Usage
	if err := inv.decode(http.MethodGet, "/company/usage", &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// SyncReceived asks the SEFAZ now for the documents issued against this
// company since the last read, instead of waiting for the scheduled
// collection, which runs every six hours. After a read that finds nothing
// more the SEFAZ makes the CNPJ wait an hour; a call inside that wait is a
// *RateLimitedError with Code DISTRIBUTION_COOLDOWN, RetryAt and
// RetryAfter.
func (inv *Invoice) SyncReceived() (*SyncResult, error) {
	var result SyncResult
	if err := inv.decode(http.MethodPost, "/received-invoices/sync", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (inv *transport) decode(method, path string, target any) error {
	raw, err := inv.send(method, path, nil, nil)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return &InvoiceError{Message: "unexpected response shape: empty body"}
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return &InvoiceError{Message: "unexpected response shape: " + err.Error()}
	}
	return nil
}
