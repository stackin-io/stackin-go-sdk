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
	"regexp"
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

// transport is where an API key becomes an HTTP call. Invoice, Clients,
// FiscalReference and Taxpayer all embed it, so one option type
// configures every entry point and one code path issues every request.
type transport struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration

	// Country is read by FiscalReference and Taxpayer only. Invoice
	// embeds it and ignores it: a fiscal document carries its issuer's
	// country already.
	Country string

	// Company is sent as X-Stackin-Company on every request when set.
	Company string

	httpClient *http.Client
}

type Invoice struct {
	transport
}

type Option func(*transport)

func WithBaseURL(baseURL string) Option {
	return func(t *transport) { t.BaseURL = baseURL }
}

func WithEnvironment(env Environment) Option {
	return func(t *transport) { t.BaseURL = environmentURLs[env] }
}

func WithAPIKey(apiKey string) Option {
	return func(t *transport) { t.APIKey = apiKey }
}

func WithTimeout(timeout time.Duration) Option {
	return func(t *transport) { t.Timeout = timeout }
}

// WithCountry sets the ISO 3166-1 alpha-2 code FiscalReference and
// Taxpayer look codes up under. Defaults to BR.
func WithCountry(country string) Option {
	return func(t *transport) { t.Country = country }
}

// WithCompany names the company every request acts on, by its id (UUID)
// or its CNPJ, punctuated or not. It is sent as the X-Stackin-Company
// header. Only a credential covering several companies needs it; an API
// key is one company already. Left empty, nothing is sent.
func WithCompany(company string) Option {
	return func(t *transport) { t.Company = company }
}

func newTransport(opts ...Option) transport {
	t := transport{
		BaseURL: resolveBaseURL(),
		APIKey:  os.Getenv("STACKIN_API_KEY"),
		Timeout: 30 * time.Second,
		Country: "BR",
	}
	for _, opt := range opts {
		opt(&t)
	}
	t.BaseURL = strings.TrimRight(t.BaseURL, "/")
	t.httpClient = &http.Client{Timeout: t.Timeout}
	return t
}

func NewInvoice(opts ...Option) *Invoice {
	return &Invoice{transport: newTransport(opts...)}
}

// ForCompany returns a copy of the client bound to another company, for
// the calls chained on it: inv.ForCompany("00.000.000/0001-91").Issue(req).
// The receiver is left as it was and the copy shares its HTTP client. An
// empty company sends no header at all.
func (inv *Invoice) ForCompany(company string) *Invoice {
	bound := *inv
	bound.Company = company
	return &bound
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

	// ExternalReference is the caller's own id for the document, an order
	// number say. The API stores it as sent (at most 255 characters) and
	// returns it on every read and webhook. Omitted when empty.
	ExternalReference string

	// IdempotencyKey makes a retry safe: the same key with the same body
	// replays the first response instead of issuing a second document.
	// Keys live 24 hours. Never generated for you — only the caller knows
	// which two requests are meant to be the same one.
	IdempotencyKey string

	// Discount is a discount on the whole document, in BRL, zero or more.
	// The API spreads it over the lines. Omitted when nil.
	Discount *float64

	// IssuedAt is the emission moment (dhEmi), sent as RFC 3339 in the
	// offset of its own Location. NF-e: at most 30 days back and 5 minutes
	// ahead; NFS-e: never ahead. The API defaults it to now. Omitted when
	// nil.
	IssuedAt *time.Time

	// Purpose is why the NF-e is issued; Direction whether it is outgoing
	// or incoming. NF-e only, omitted when empty.
	Purpose   Purpose
	Direction Direction

	// ReferencedKeys are the 44-character access keys of the NF-e, NFC-e
	// or CF-e this NF-e refers to: required for PurposeComplementary and,
	// but for a few CFOPs, PurposeReturn. NF-e only, omitted when empty.
	ReferencedKeys []string

	// ForeignID identifies a buyer abroad (idEstrangeiro, 5 to 20
	// characters), for an NF-e whose RecipientAddress.State is EX. TaxID
	// is then left out. Omitted when empty.
	ForeignID string

	// Export is where the goods leave Brazil, required for an outgoing
	// NF-e to a buyer abroad. Omitted when nil.
	Export *Export

	// ClientID names a saved client of the company (see Clients). Its
	// name, tax id, foreign id and address fill whichever of ClientName,
	// TaxID, ForeignID and RecipientAddress are left empty; the ones set
	// win, and the empty ones are left out of the request. The address
	// checks then apply only to a RecipientAddress that is set: the API
	// checks the merged request. Omitted when empty.
	ClientID string

	// UpdateClient rewrites an existing client with this request's
	// recipient data after a successful issuance; a new recipient is
	// saved as a client either way. Sent only when true.
	UpdateClient bool
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

// missingAddressFields names the empty required fields of address, in
// declaration order. CityCode is never among them: the API resolves it from
// City and State. When abroad, ZipCode gives way to CountryCode.
func missingAddressFields(address *Address, abroad bool) []string {
	type field struct {
		name  string
		value string
	}
	fields := []field{
		{"Street", address.Street},
		{"Number", address.Number},
		{"Neighborhood", address.Neighborhood},
		{"City", address.City},
		{"State", address.State},
	}
	if abroad {
		fields = append(fields, field{"CountryCode", address.CountryCode})
	} else {
		fields = append(fields, field{"ZipCode", address.ZipCode})
	}

	var missing []string
	for _, field := range fields {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}

// validateRetainedISSAddress applies to an NFSE with any TaxRetained item:
// the city authorizer refuses it without the tomador's address (E0237).
// Every field is required except CityCode.
func validateRetainedISSAddress(req IssueRequest) error {
	retained := false
	for _, item := range req.Items {
		if item.TaxRetained {
			retained = true
			break
		}
	}
	if !retained {
		return nil
	}
	if req.RecipientAddress == nil {
		return &InvoiceError{Message: "recipient_address is required on an NFS-e " +
			"when any item has tax_retained=true (ISSQN retained by the tomador)"}
	}
	if missing := missingAddressFields(req.RecipientAddress, false); len(missing) > 0 {
		return &InvoiceError{Message: fmt.Sprintf(
			"RecipientAddress is missing required fields for NFSE with tax_retained=true: %s",
			strings.Join(missing, ", "),
		)}
	}
	return nil
}

// validateNFEAddress rejects a missing or partial buyer address before the
// network call — the SEFAZ rejects a partial enderDest (274/726/696/695).
// Every field is required except CityCode, which the API resolves from City
// and State, answering CITY_NOT_FOUND (422) when it cannot. A buyer abroad
// (State EX) has no ZipCode and needs a CountryCode instead.
func validateNFEAddress(address *Address) error {
	if address == nil {
		return &InvoiceError{Message: "RecipientAddress is required for NFE"}
	}

	if missing := missingAddressFields(address, address.abroad()); len(missing) > 0 {
		return &InvoiceError{Message: fmt.Sprintf(
			"RecipientAddress is missing required fields for NFE: %s",
			strings.Join(missing, ", "),
		)}
	}

	return nil
}

// apiConvertedRunes are the characters above U+00FF the API rewrites to a
// Latin-1 spelling before building the NF-e (quotes, dashes, ellipsis,
// bullet, euro and trademark signs). The SDK sends them as they are.
var apiConvertedRunes = map[rune]bool{
	'\u2018': true, '\u2019': true, '\u201a': true, // ‘ ’ ‚ -> '
	'\u201c': true, '\u201d': true, '\u201e': true, // “ ” „ -> "
	'\u2013': true, '\u2014': true, '\u2212': true, // – — − -> -
	'\u2026': true, // … -> ...
	'\u2022': true, // • -> -
	'\u20ac': true, // € -> EUR
	'\u2122': true, // ™ -> TM
}

// validateLatin1 rejects the first free-text field of an NF-e payload
// holding a character the NF-e cannot carry: anything above U+00FF
// (ISO-8859-1) that the API does not convert itself. Whitespace such as
// tabs and newlines is collapsed by the API and passes. Paths name the
// payload keys sent.
func validateLatin1(req IssueRequest) error {
	type field struct {
		path  string
		value string
	}
	fields := []field{{"client_name", req.ClientName}}
	for i, item := range req.Items {
		prefix := fmt.Sprintf("items[%d].", i)
		fields = append(fields,
			field{prefix + "description", item.Description},
			field{prefix + "product.unit", item.Unit},
			field{prefix + "product.purchase_order", deref(item.PurchaseOrder)},
			field{prefix + "product.purchase_order_item", deref(item.PurchaseOrderItem)},
			field{prefix + "observations", deref(item.Observations)},
		)
		if fuel := item.Fuel; fuel != nil {
			fields = append(fields,
				field{prefix + "product.br.fuel.anp_description", fuel.AnpDescription},
				field{prefix + "product.br.fuel.consumer_state", fuel.ConsumerState},
				field{prefix + "product.br.fuel.codif", deref(fuel.Codif)},
			)
		}
	}
	if a := req.RecipientAddress; a != nil {
		fields = append(fields,
			field{"recipient_address.street", a.Street},
			field{"recipient_address.number", a.Number},
			field{"recipient_address.neighborhood", a.Neighborhood},
			field{"recipient_address.city", a.City},
			field{"recipient_address.state", a.State},
			field{"recipient_address.zip_code", a.ZipCode},
			field{"recipient_address.city_code", a.CityCode},
			field{"recipient_address.country_code", a.CountryCode},
			field{"recipient_address.country_name", a.CountryName},
		)
	}
	fields = append(fields, field{"foreign_id", req.ForeignID})
	if e := req.Export; e != nil {
		fields = append(fields,
			field{"export.exit_state", e.ExitState},
			field{"export.place", e.Place},
			field{"export.dispatch_place", e.DispatchPlace},
		)
	}
	for _, f := range fields {
		for _, r := range f.value {
			if r > 0xFF && !apiConvertedRunes[r] {
				return &InvoiceError{Message: fmt.Sprintf(
					"%s contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '%c'",
					f.path, r,
				)}
			}
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// validateIssueFields checks the optional request fields whose shape is
// unambiguous: a negative Discount, a zero IssuedAt and a Purpose or
// Direction outside its enumeration. Everything else about them is a
// fiscal rule the API owns.
func validateIssueFields(req IssueRequest) error {
	if req.Discount != nil && *req.Discount < 0 {
		return &InvoiceError{Message: "discount must be zero or more"}
	}
	if req.IssuedAt != nil && req.IssuedAt.IsZero() {
		return &InvoiceError{Message: "issued_at must be a date-time, not the zero time"}
	}
	switch req.Purpose {
	case "", PurposeNormal, PurposeComplementary, PurposeAdjustment, PurposeReturn:
	default:
		return &InvoiceError{Message: fmt.Sprintf(
			"purpose must be normal, complementary, adjustment or return, not %q", req.Purpose,
		)}
	}
	switch req.Direction {
	case "", DirectionOut, DirectionIn:
	default:
		return &InvoiceError{Message: fmt.Sprintf("direction must be out or in, not %q", req.Direction)}
	}
	return nil
}

// nfeDescriptionLimit is xProd's maximum length in the NF-e 4.00 schema.
// NFS-e descriptions are not bound by it.
const nfeDescriptionLimit = 120

func (inv *Invoice) Issue(req IssueRequest) (map[string]any, error) {
	if len(req.Items) == 0 {
		return nil, &InvoiceError{Message: "items can't be empty"}
	}

	if err := validateIssueFields(req); err != nil {
		return nil, err
	}

	for i, item := range req.Items {
		if err := validateProduct(item); err != nil {
			return nil, &InvoiceError{Message: fmt.Sprintf("items[%d].%s", i, err.Error())}
		}
	}
	if req.RecipientAddress != nil {
		if err := req.RecipientAddress.validate(); err != nil {
			return nil, &InvoiceError{Message: "recipient_address." + err.Error()}
		}
	}

	if req.DocumentType == NFE {
		for i, item := range req.Items {
			if item.NCM == nil || *item.NCM == "" {
				return nil, &InvoiceError{Message: fmt.Sprintf("items[%d].ncm is required for NFE", i)}
			}
			if item.CFOP == nil || *item.CFOP == "" {
				return nil, &InvoiceError{Message: fmt.Sprintf("items[%d].cfop is required for NFE", i)}
			}
			if len([]rune(item.Description)) > nfeDescriptionLimit {
				return nil, &InvoiceError{Message: fmt.Sprintf(
					"items[%d].description exceeds %d characters (NF-e xProd limit)", i, nfeDescriptionLimit,
				)}
			}
		}
		if req.RecipientAddress != nil || req.ClientID == "" {
			if err := validateNFEAddress(req.RecipientAddress); err != nil {
				return nil, err
			}
		}
		if err := validateLatin1(req); err != nil {
			return nil, err
		}
	}
	if req.DocumentType == NFSE && (req.RecipientAddress != nil || req.ClientID == "") {
		if err := validateRetainedISSAddress(req); err != nil {
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
	if req.TaxID == "" && req.DocumentType == NFE && req.RecipientAddress.abroad() {
		delete(payload, "tax_id")
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
		if req.ClientName == "" {
			delete(payload, "client_name")
		}
		if req.TaxID == "" {
			delete(payload, "tax_id")
		}
	}
	if req.UpdateClient {
		payload["update_client"] = true
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
	if req.ExternalReference != "" {
		payload["external_reference"] = req.ExternalReference
	}
	if req.Discount != nil {
		payload["discount"] = strconv.FormatFloat(*req.Discount, 'f', -1, 64)
	}
	if req.IssuedAt != nil {
		payload["issued_at"] = req.IssuedAt.Format(time.RFC3339)
	}
	if req.Purpose != "" {
		payload["purpose"] = string(req.Purpose)
	}
	if req.Direction != "" {
		payload["direction"] = string(req.Direction)
	}
	if len(req.ReferencedKeys) > 0 {
		payload["referenced_keys"] = req.ReferencedKeys
	}
	if req.ForeignID != "" {
		payload["foreign_id"] = req.ForeignID
	}
	if req.Export != nil {
		payload["export"] = req.Export
	}

	return inv.request(http.MethodPost, "/invoices", payload, nil,
		WithIdempotencyKey(req.IdempotencyKey))
}

func (inv *Invoice) Consult(accessKey string, documentType DocumentType) (map[string]any, error) {
	params := url.Values{"document_type": {string(documentType)}}
	return inv.request(http.MethodGet, "/invoices/"+accessKey, nil, params)
}

// Cancel voids an authorized document. Pass WithIdempotencyKey to make a
// retry safe: repeating the same key with the same body replays the first
// answer instead of cancelling twice. This is the one irreversible
// operation here, with a legal window and no undo.
func (inv *Invoice) Cancel(accessKey string, documentType DocumentType, reason string, opts ...RequestOption) (map[string]any, error) {
	payload := map[string]any{
		"document_type": string(documentType),
		"reason":        reason,
	}
	return inv.request(http.MethodPost, "/invoices/"+accessKey+"/cancel", payload, nil, opts...)
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

// nfeAccessKeyPattern is the shape of an NF-e access key (44 characters).
// Positions 7-18 carry the start of the issuer's CNPJ, which may be
// alphanumeric (IN RFB 2.229/2024); the rest of the key is numeric. An
// NFS-e key is 50 digits and never matches.
var nfeAccessKeyPattern = regexp.MustCompile(`^[0-9]{6}[0-9A-Z]{12}[0-9]{26}$`)

// Correct files a correction letter (CC-e). NF-e only: a key that is not an
// NF-e access key is refused before the call. A well-formed key sent with
// NFSE, or one for a document the API does not hold, is answered 409 by the
// API and comes back as an *APIError.
func (inv *Invoice) Correct(accessKey string, documentType DocumentType, correction string) (map[string]any, error) {
	if !nfeAccessKeyPattern.MatchString(accessKey) {
		return nil, &InvoiceError{
			Message: "Correct is NFe only. NFSe has no correction letter — cancel and reissue instead.",
		}
	}
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

// HistoryQuery filters the issuance history. Every field is optional; the
// zero value asks for the newest page with the API's own defaults.
type HistoryQuery struct {
	DocumentType DocumentType
	Status       string
	Limit        int
	Offset       int
	SortBy       string
	OrderBy      string
}

// History lists what this company issued, newest first.
//
// The counterpart of Received: that one lists what was issued against the
// company, this one what the company issued.
func (inv *Invoice) History(query HistoryQuery) (map[string]any, error) {
	params := url.Values{}
	if query.DocumentType != "" {
		params.Set("document_type", string(query.DocumentType))
	}
	if query.Status != "" {
		params.Set("status", query.Status)
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
	return inv.request(http.MethodGet, "/invoices", nil, params)
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

// Submissions returns every attempt made for one invoice, with what the
// authorizer answered to each.
//
// Consult gives the status; this gives the reason. It takes the invoiceID,
// like Reissue and unlike everything else, because a rejected document has
// no access key to look it up by.
func (inv *Invoice) Submissions(invoiceID string) ([]map[string]any, error) {
	raw, err := inv.send(http.MethodGet, "/invoices/"+invoiceID+"/submissions", nil, nil)
	if err != nil {
		return nil, err
	}

	var parsed []map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
		}
	}
	return parsed, nil
}

// AuthorizedCompany is one company a credential may act on, and the
// permissions it holds there.
type AuthorizedCompany struct {
	CompanyID   string   `json:"company_id"`
	Name        string   `json:"name"`
	TaxID       string   `json:"tax_id"`
	Permissions []string `json:"permissions"`
}

// AuthorizedCompanies is every company a credential covers, in one
// environment.
type AuthorizedCompanies struct {
	Environment string              `json:"environment"`
	Companies   []AuthorizedCompany `json:"companies"`
}

// Companies lists the companies this credential reaches, with the
// permissions it holds on each: the ids WithCompany and ForCompany
// accept. An API key lists as its one company with every permission.
func (inv *Invoice) Companies() (*AuthorizedCompanies, error) {
	raw, err := inv.send(http.MethodGet, "/me/companies", nil, nil)
	if err != nil {
		return nil, err
	}

	var parsed AuthorizedCompanies
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
		}
	}
	return &parsed, nil
}

func (inv *transport) request(method, path string, payload any, params url.Values, opts ...RequestOption) (map[string]any, error) {
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

func (inv *transport) send(method, path string, payload any, params url.Values, opts ...RequestOption) ([]byte, error) {
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
	if inv.Company != "" {
		httpReq.Header.Set("X-Stackin-Company", inv.Company)
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
		var structured json.RawMessage
		if d, ok := parsed["detail"]; ok {
			if s, ok := d.(string); ok {
				detail = s
			} else if b, err := json.Marshal(d); err == nil {
				detail = string(b)
				structured = b
			}
		}
		return nil, newAPIError(resp.StatusCode, detail, structured, raw, resp.Header)
	}

	return raw, nil
}
