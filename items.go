package stackin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/stackin-io/stackin-go-sdk/br"
)

// priceText matches the plain decimal a catalog price may be sent as: up to
// 10 places, the vUnCom precision.
var priceText = regexp.MustCompile(`^\d+(\.\d{1,10})?$`)

// priceWire gives the decimal string a unit price goes out as. The text
// wins over the float, and is checked; the float is written as a plain
// decimal, never in exponent form. Both unset gives "" (not sent).
func priceWire(number *float64, text string) (string, error) {
	if text != "" {
		if !priceText.MatchString(text) {
			return "", &InvoiceError{Message: "unit price must be a decimal with at most 10 places, such as 18.90"}
		}
		return text, nil
	}
	if number != nil {
		return br.DecimalString(*number), nil
	}
	return "", nil
}

// Items reads and edits the company's catalog of products and services,
// and Product.ItemID issues a line from one. The catalog is one table told
// apart by Kind: "product" for goods (NF-e) and "service" for services
// (NFS-e).
//
// It is company data, like everything on Invoice: WithCompany and
// ForCompany pick the company. Reading needs invoice:read; creating,
// editing and deactivating need invoice:issue. The rules on an item's
// fields are the API's: a refusal comes back as an *APIError 422.
type Items struct {
	transport
}

func NewItems(opts ...Option) *Items {
	return &Items{transport: newTransport(opts...)}
}

// ForCompany returns a copy of the client bound to another company, for
// the calls chained on it. The receiver is left as it was.
func (i *Items) ForCompany(company string) *Items {
	bound := *i
	bound.Company = company
	return &bound
}

// ItemQuery narrows the item listing. Every field is optional; the zero
// value asks for the first page of active items, by description.
type ItemQuery struct {
	// Kind is product or service.
	Kind string
	// Search matches part of the description, code or external reference;
	// digits also match the NCM and the GTIN.
	Search string
	// Source is manual, import, nuvemshop or invoice.
	Source string
	// IsActive left nil lists the active items, as the API does; pointing
	// at false lists the deactivated ones.
	IsActive *bool
	Limit    int
	Offset   int
	// SortBy takes description, code, unit_price, last_used_at,
	// created_at or uses_count.
	SortBy  string
	OrderBy string
}

// ItemFields is an item as Create and Upsert take it. Kind and
// Description are required. The NCM, CFOP, TaxSituation, Origin, CEST and
// GTIN fields belong to a product and ServiceCode and ISSRetained to a
// service; the other kind's is a 422. Empty text fields and nil pointers
// are left out of the request; UnitPrice and ISSRetained are pointers so a
// price of 0 and ISSRetained false (turning retention off) can be sent.
type ItemFields struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
	// UnitPrice is a pointer so a price of 0 can be sent, and goes out as a
	// decimal string. UnitPriceText (such as "18.9000000001", up to 10
	// places) wins over it and keeps every digit a float64 would round.
	UnitPrice         *float64 `json:"-"`
	UnitPriceText     string   `json:"-"`
	Unit              string   `json:"unit,omitempty"`
	ExternalReference string   `json:"external_reference,omitempty"`
	Notes             string   `json:"notes,omitempty"`
	// Source is how the item came to be (manual, import, nuvemshop or
	// invoice). Upsert only, and only when it creates; the API defaults it
	// to manual.
	Source       string `json:"source,omitempty"`
	NCM          string `json:"ncm,omitempty"`
	CFOP         string `json:"cfop,omitempty"`
	TaxSituation string `json:"tax_situation,omitempty"`
	Origin       string `json:"origin,omitempty"`
	CEST         string `json:"cest,omitempty"`
	GTIN         string `json:"gtin,omitempty"`
	ServiceCode  string `json:"service_code,omitempty"`
	ISSRetained  *bool  `json:"iss_retained,omitempty"`
}

// MarshalJSON writes the unit price as a decimal string.
func (f ItemFields) MarshalJSON() ([]byte, error) {
	price, err := priceWire(f.UnitPrice, f.UnitPriceText)
	if err != nil {
		return nil, err
	}
	type plain ItemFields
	out := struct {
		plain
		UnitPrice string `json:"unit_price,omitempty"`
	}{plain: plain(f), UnitPrice: price}
	return json.Marshal(out)
}

// ItemChanges is what Update changes. A nil field is not sent; a text
// field pointing at "" is sent as null, which clears it. The API checks
// the item as a whole, so changing Kind while a field of the old kind is
// still set is a 422.
type ItemChanges struct {
	Kind        *string
	Description *string
	Code        *string
	// UnitPrice is sent as a decimal string. UnitPriceText (up to 10
	// places) wins over it and keeps every digit; pointing it at "" clears
	// the price.
	UnitPrice         *float64
	UnitPriceText     *string
	Unit              *string
	ExternalReference *string
	Notes             *string
	NCM               *string
	CFOP              *string
	TaxSituation      *string
	Origin            *string
	CEST              *string
	GTIN              *string
	ServiceCode       *string
	ISSRetained       *bool
	// IsActive pointing at true brings a deactivated item back; see
	// Reactivate.
	IsActive *bool
}

func (changes ItemChanges) body() (map[string]any, error) {
	body := map[string]any{}
	texts := []struct {
		key   string
		value *string
	}{
		{"kind", changes.Kind},
		{"description", changes.Description},
		{"code", changes.Code},
		{"unit", changes.Unit},
		{"external_reference", changes.ExternalReference},
		{"notes", changes.Notes},
		{"ncm", changes.NCM},
		{"cfop", changes.CFOP},
		{"tax_situation", changes.TaxSituation},
		{"origin", changes.Origin},
		{"cest", changes.CEST},
		{"gtin", changes.GTIN},
		{"service_code", changes.ServiceCode},
	}
	for _, text := range texts {
		switch {
		case text.value == nil:
		case *text.value == "":
			body[text.key] = nil
		default:
			body[text.key] = *text.value
		}
	}
	switch {
	case changes.UnitPriceText != nil && *changes.UnitPriceText == "":
		body["unit_price"] = nil
	case changes.UnitPriceText != nil:
		price, err := priceWire(nil, *changes.UnitPriceText)
		if err != nil {
			return nil, err
		}
		body["unit_price"] = price
	case changes.UnitPrice != nil:
		body["unit_price"] = br.DecimalString(*changes.UnitPrice)
	}
	if changes.ISSRetained != nil {
		body["iss_retained"] = *changes.ISSRetained
	}
	if changes.IsActive != nil {
		body["is_active"] = *changes.IsActive
	}
	return body, nil
}

// List returns a page of the company's items.
func (i *Items) List(query ItemQuery) (map[string]any, error) {
	params := url.Values{}
	if query.Kind != "" {
		params.Set("kind", query.Kind)
	}
	if query.Search != "" {
		params.Set("search", query.Search)
	}
	if query.Source != "" {
		params.Set("source", query.Source)
	}
	if query.IsActive != nil {
		params.Set("is_active", strconv.FormatBool(*query.IsActive))
	}
	if query.Limit > 0 {
		params.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Offset > 0 {
		params.Set("offset", strconv.Itoa(query.Offset))
	}
	if query.SortBy != "" {
		params.Set("sort_by", query.SortBy)
	}
	if query.OrderBy != "" {
		params.Set("order_by", query.OrderBy)
	}
	return i.request(http.MethodGet, "/items", nil, params)
}

// Get returns one item, deactivated or not. Another company's item is a
// 404 (*ItemNotFoundError).
func (i *Items) Get(itemID string) (map[string]any, error) {
	escaped, err := segment(itemID)
	if err != nil {
		return nil, err
	}
	return i.request(http.MethodGet, "/items/"+escaped, nil, nil)
}

// Upsert saves an item by its ExternalReference, or by its Code when no
// reference is sent: the company's matching item is updated with the
// fields sent (it keeps its source and its active flag), otherwise a new
// one is created. created is true for the API's 201 and false for its 200,
// so sending the same fields twice never makes two items. A Code the
// company holds on another item is a *ItemAlreadyExistsError.
func (i *Items) Upsert(fields ItemFields) (item map[string]any, created bool, err error) {
	raw, status, err := i.sendStatus(http.MethodPost, "/items", fields, nil)
	if err != nil {
		return nil, false, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &item)
	}
	if result, ok := item["result"].(map[string]any); ok {
		item = result
	}
	return item, status == http.StatusCreated, nil
}

// Create is Upsert without the created flag.
func (i *Items) Create(fields ItemFields) (map[string]any, error) {
	item, _, err := i.Upsert(fields)
	return item, err
}

// Update changes only the fields set in changes. The API checks the item
// as a whole; taking a code another item holds is a
// *ItemAlreadyExistsError.
func (i *Items) Update(itemID string, changes ItemChanges) (map[string]any, error) {
	escaped, err := segment(itemID)
	if err != nil {
		return nil, err
	}
	body, err := changes.body()
	if err != nil {
		return nil, err
	}
	return i.request(http.MethodPatch, "/items/"+escaped, body, nil)
}

// Deactivate deactivates an item. It is kept, still holds its code, is
// left out of List by default and refused on issue until Reactivate brings
// it back. The API answers 204 with no body.
func (i *Items) Deactivate(itemID string) error {
	escaped, err := segment(itemID)
	if err != nil {
		return err
	}
	_, err = i.send(http.MethodDelete, "/items/"+escaped, nil, nil)
	return err
}

// Reactivate brings a deactivated item back, and returns it.
func (i *Items) Reactivate(itemID string) (map[string]any, error) {
	active := true
	return i.Update(itemID, ItemChanges{IsActive: &active})
}
