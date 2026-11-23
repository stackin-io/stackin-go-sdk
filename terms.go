package stackin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var (
	presenceCodes        = codeSet("0", "1", "2", "3", "4", "5", "9")
	freightModeCodes     = codeSet("0", "1", "2", "3", "4", "9")
	paymentMethodPattern = regexp.MustCompile(`^\d{2}$`)
)

func codeSet(codes ...string) map[string]bool {
	set := make(map[string]bool, len(codes))
	for _, code := range codes {
		set[code] = true
	}
	return set
}

// Intermediary is the marketplace or platform an NF-e sale went through
// (infIntermed). TaxID is its CNPJ and Identifier the seller's user name or
// profile id there.
type Intermediary struct {
	TaxID      string `json:"tax_id"`
	Identifier string `json:"identifier"`
}

// TransportVolume is one transp/vol group. Every field is optional and sent only
// when set; the pointers let a real 0 go out. Weights are in kg with at
// most 3 decimal places.
type TransportVolume struct {
	Quantity    *int     `json:"quantity,omitempty"`
	Species     string   `json:"species,omitempty"`
	Brand       string   `json:"brand,omitempty"`
	Numbering   string   `json:"numbering,omitempty"`
	NetWeight   *float64 `json:"net_weight,omitempty"`
	GrossWeight *float64 `json:"gross_weight,omitempty"`
}

// Transport is the freight mode (modFrete) and the volumes carried.
// Volumes left nil lets the API derive one from the lines' weights; an
// empty non-nil slice is sent as [] and writes none.
type Transport struct {
	FreightMode string
	Volumes     []TransportVolume
}

func (t Transport) MarshalJSON() ([]byte, error) {
	out := map[string]any{"freight_mode": t.FreightMode}
	if t.Volumes != nil {
		out["volumes"] = t.Volumes
	}
	return json.Marshal(out)
}

func (t *Transport) UnmarshalJSON(data []byte) error {
	var raw struct {
		FreightMode string          `json:"freight_mode"`
		Volumes     json.RawMessage `json:"volumes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.FreightMode = raw.FreightMode
	t.Volumes = nil
	if len(raw.Volumes) > 0 && string(raw.Volumes) != "null" {
		t.Volumes = []TransportVolume{}
		return json.Unmarshal(raw.Volumes, &t.Volumes)
	}
	return nil
}

func (v *TransportVolume) UnmarshalJSON(data []byte) error {
	var raw struct {
		Quantity    *int            `json:"quantity"`
		Species     string          `json:"species"`
		Brand       string          `json:"brand"`
		Numbering   string          `json:"numbering"`
		NetWeight   json.RawMessage `json:"net_weight"`
		GrossWeight json.RawMessage `json:"gross_weight"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	net, err := decimalFromJSON(raw.NetWeight)
	if err != nil {
		return fmt.Errorf("net_weight: %w", err)
	}
	gross, err := decimalFromJSON(raw.GrossWeight)
	if err != nil {
		return fmt.Errorf("gross_weight: %w", err)
	}
	*v = TransportVolume{
		Quantity: raw.Quantity, Species: raw.Species, Brand: raw.Brand,
		Numbering: raw.Numbering, NetWeight: net, GrossWeight: gross,
	}
	return nil
}

// PaymentCard is the electronic payment's processor and authorization
// (card). Integration is 1 (integrated with the seller's system) or 2.
type PaymentCard struct {
	Integration   string `json:"integration"`
	TaxID         string `json:"tax_id,omitempty"`
	Brand         string `json:"brand,omitempty"`
	Authorization string `json:"authorization,omitempty"`
}

// Payment is one detPag. Method is the national tPag code, such as "01" or
// "17"; Amount is in BRL and is always sent, so a 0 (method "90") goes out.
// Description is required for method "99" and sent only then.
type Payment struct {
	Method      string
	Amount      float64
	Description string
	Card        *PaymentCard
}

func (p Payment) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Method      string       `json:"method"`
		Amount      string       `json:"amount"`
		Description string       `json:"description,omitempty"`
		Card        *PaymentCard `json:"card,omitempty"`
	}{p.Method, br.DecimalString(p.Amount), p.Description, p.Card})
}

func (p *Payment) UnmarshalJSON(data []byte) error {
	var raw struct {
		Method      string          `json:"method"`
		Amount      json.RawMessage `json:"amount"`
		Description string          `json:"description"`
		Card        *PaymentCard    `json:"card"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	amount, err := decimalFromJSON(raw.Amount)
	if err != nil {
		return fmt.Errorf("amount: %w", err)
	}
	*p = Payment{Method: raw.Method, Description: raw.Description, Card: raw.Card}
	if amount != nil {
		p.Amount = *amount
	}
	return nil
}

// SaleTerms are the NF-e sale terms an invoice was issued with, as the
// invoice detail returns them. Every field is nil or empty when the
// issuance did not send it. Change is in BRL.
type SaleTerms struct {
	Presence     *string       `json:"presence"`
	Intermediary *Intermediary `json:"intermediary"`
	Transport    *Transport    `json:"transport"`
	Payments     []Payment     `json:"payments"`
	Change       *float64      `json:"-"`
}

func (s *SaleTerms) UnmarshalJSON(data []byte) error {
	type plain SaleTerms
	var raw struct {
		plain
		Change json.RawMessage `json:"change"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	change, err := decimalFromJSON(raw.Change)
	if err != nil {
		return fmt.Errorf("change: %w", err)
	}
	*s = SaleTerms(raw.plain)
	s.Change = change
	return nil
}

// SaleTermsOf reads the stored sale terms out of an invoice detail, as
// returned by the API: presence, intermediary, transport, payments and
// change at the top level of the object (or of its "result"). Fields the
// detail does not carry stay nil.
func SaleTermsOf(detail map[string]any) (*SaleTerms, error) {
	if result, ok := detail["result"].(map[string]any); ok {
		detail = result
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		return nil, &InvoiceError{Message: "unexpected invoice detail: " + err.Error()}
	}
	var terms SaleTerms
	if err := json.Unmarshal(raw, &terms); err != nil {
		return nil, &InvoiceError{Message: "unexpected invoice detail: " + err.Error()}
	}
	return &terms, nil
}

func decimalFromJSON(raw json.RawMessage) (*float64, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return nil, nil
	}
	text = strings.Trim(text, `"`)
	var value float64
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return nil, fmt.Errorf("%q is not a decimal", text)
	}
	return &value, nil
}

func decimalPlaces(value float64) int {
	text := br.DecimalString(value)
	dot := strings.IndexByte(text, '.')
	if dot < 0 {
		return 0
	}
	return len(text) - dot - 1
}

func validateTerms(req IssueRequest) error {
	if req.Presence != "" && !presenceCodes[req.Presence] {
		return &InvoiceError{Message: fmt.Sprintf(
			"presence must be 0, 1, 2, 3, 4, 5 or 9, not %q", req.Presence,
		)}
	}
	if req.Intermediary != nil {
		length := len([]rune(req.Intermediary.Identifier))
		if length < 2 || length > 60 {
			return &InvoiceError{Message: "intermediary.identifier must be 2 to 60 characters"}
		}
	}
	if req.Change != nil {
		if *req.Change < 0 {
			return &InvoiceError{Message: "change can't be negative"}
		}
		if decimalPlaces(*req.Change) > 2 {
			return &InvoiceError{Message: "change allows at most 2 decimal places"}
		}
	}
	if req.Transport != nil {
		if !freightModeCodes[req.Transport.FreightMode] {
			return &InvoiceError{Message: fmt.Sprintf(
				"transport.freight_mode must be 0, 1, 2, 3, 4 or 9, not %q", req.Transport.FreightMode,
			)}
		}
		for i, volume := range req.Transport.Volumes {
			for name, weight := range map[string]*float64{
				"net_weight": volume.NetWeight, "gross_weight": volume.GrossWeight,
			} {
				if weight == nil {
					continue
				}
				if *weight < 0 {
					return &InvoiceError{Message: fmt.Sprintf("transport.volumes[%d].%s can't be negative", i, name)}
				}
				if decimalPlaces(*weight) > 3 {
					return &InvoiceError{Message: fmt.Sprintf("transport.volumes[%d].%s allows at most 3 decimal places", i, name)}
				}
			}
		}
	}
	if req.Payments != nil && len(req.Payments) == 0 {
		return &InvoiceError{Message: "payments needs 1 to 100 entries"}
	}
	if len(req.Payments) > 100 {
		return &InvoiceError{Message: "payments needs 1 to 100 entries"}
	}
	for i, payment := range req.Payments {
		if !paymentMethodPattern.MatchString(payment.Method) {
			return &InvoiceError{Message: fmt.Sprintf(
				"payments[%d].method must be a two-digit code, not %q", i, payment.Method,
			)}
		}
		if payment.Amount < 0 {
			return &InvoiceError{Message: fmt.Sprintf("payments[%d].amount can't be negative", i)}
		}
		if decimalPlaces(payment.Amount) > 2 {
			return &InvoiceError{Message: fmt.Sprintf("payments[%d].amount allows at most 2 decimal places", i)}
		}
		if payment.Method == "99" && strings.TrimSpace(payment.Description) == "" {
			return &InvoiceError{Message: fmt.Sprintf(
				"payments[%d].description is required for method 99", i,
			)}
		}
	}
	return nil
}
