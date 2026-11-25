package stackin

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// segment escapes one path segment, refusing the ones that would leave
// it. url.PathEscape leaves "." alone — RFC 3986 calls it unreserved —
// so the dot segments have to be refused rather than escaped, or the
// HTTP client collapses ".." into a different endpoint.
func segment(value string) (string, error) {
	if value == "" || value == "." || value == ".." {
		return "", &InvoiceError{Message: fmt.Sprintf("%q is not a usable path segment", value)}
	}
	return url.PathEscape(value), nil
}

// Kinds are the classifications with a named accessor on
// FiscalReference. It is not the whole truth and is not meant to be:
// ask Kinds() for what the API actually has, because a classification
// published after this release is reachable through Kind() with no
// release at all.
var Kinds = []string{
	"cfop",
	"ncm",
	"cest",
	"cst",
	"csosn",
	"iss_service",
	"icms_fuel",
	"ibs_cbs_class",
}

// SearchQuery narrows a classification listing.
//
// There is no SortBy or OrderBy: the route accepts both and discards
// them, so rows always come back ordered by kind then code. History on
// Invoice does honour them, which is why the difference is spelled out
// here rather than left to be discovered.
type SearchQuery struct {
	Term    string
	Country string
	Limit   int
	Offset  int
}

// Kind is one classification, bound to a country.
type Kind struct {
	client  *transport
	Name    string
	Country string
}

// Get returns one code. A code that does not exist is a 404, which
// arrives as *APIError — there is no separate not-found type.
func (k *Kind) Get(code string, country ...string) (map[string]any, error) {
	name, err := segment(k.Name)
	if err != nil {
		return nil, err
	}
	escaped, err := segment(code)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("country", pickCountry(k.Country, country))

	return k.client.request("GET", "/fiscal-references/"+name+"/"+escaped, nil, params)
}

// Search returns a page of this classification, filtered by the query's
// term when it has one. An empty term is no term: the route rejects
// search="" with a 422, and sending it would invent a failure the
// caller cannot read.
func (k *Kind) Search(query SearchQuery) (map[string]any, error) {
	return searchReferences(k.client, k.Name, k.Country, query)
}

// FiscalReference reads the published classification tables. Nothing
// here is the company's own data and nothing here is writable.
type FiscalReference struct {
	transport

	CFOP        *Kind
	NCM         *Kind
	CEST        *Kind
	CST         *Kind
	CSOSN       *Kind
	IssService  *Kind
	IcmsFuel    *Kind
	IbsCbsClass *Kind
}

func NewFiscalReference(opts ...Option) *FiscalReference {
	r := &FiscalReference{transport: newTransport(opts...)}

	r.CFOP = r.Kind("cfop")
	r.NCM = r.Kind("ncm")
	r.CEST = r.Kind("cest")
	r.CST = r.Kind("cst")
	r.CSOSN = r.Kind("csosn")
	r.IssService = r.Kind("iss_service")
	r.IcmsFuel = r.Kind("icms_fuel")
	r.IbsCbsClass = r.Kind("ibs_cbs_class")

	return r
}

// Kinds asks which classifications this country has rows for. The only
// honest answer to "what else is there" — a hard-coded list goes stale
// the next time the ETL grows one.
func (r *FiscalReference) Kinds(country ...string) ([]string, error) {
	params := url.Values{}
	params.Set("country", pickCountry(r.Country, country))

	raw, err := r.send("GET", "/fiscal-references/kinds", nil, params)
	if err != nil {
		return nil, err
	}
	return decodeStrings(raw)
}

// Kind reaches any classification by name, including one with no
// accessor on this struct.
func (r *FiscalReference) Kind(name string, country ...string) *Kind {
	return &Kind{
		client:  &r.transport,
		Name:    name,
		Country: pickCountry(r.Country, country),
	}
}

// Search spans every classification at once, which no accessor can
// express. It is the most expensive call the endpoint accepts: the
// whole country's tables, not one of them.
func (r *FiscalReference) Search(query SearchQuery) (map[string]any, error) {
	return searchReferences(&r.transport, "", r.Country, query)
}

func searchReferences(client *transport, kind, fallback string, query SearchQuery) (map[string]any, error) {
	params := url.Values{}
	params.Set("country", firstNonEmpty(query.Country, fallback))
	if kind != "" {
		params.Set("kind", kind)
	}
	if query.Term != "" {
		params.Set("search", query.Term)
	}
	if query.Limit > 0 {
		params.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Offset > 0 {
		params.Set("offset", strconv.Itoa(query.Offset))
	}

	return client.request("GET", "/fiscal-references", nil, params)
}

func pickCountry(fallback string, override []string) string {
	if len(override) > 0 && override[0] != "" {
		return override[0]
	}
	return fallback
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func decodeStrings(raw []byte) ([]string, error) {
	var parsed []string
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
		}
	}
	return parsed, nil
}

// CityQuery narrows a city lookup. Both fields are optional.
type CityQuery struct {
	// State is one UF, two letters.
	State string
	// Country overrides the client's own, ISO 3166-1 alpha-2.
	Country string
}

// Cities lists the municipalities whose name starts with name, with their
// code. Each row is {country, code, name, state}; for Brazil code is the
// 7-digit IBGE code an Address's CityCode takes. Issuing resolves it from
// City and State by itself, so this is for the name that matches more
// than one. The API answers at most 20 rows.
func (r *FiscalReference) Cities(name string, query CityQuery) ([]map[string]any, error) {
	length := len([]rune(name))
	if length < 2 || length > 100 {
		return nil, &InvoiceError{Message: "name must be 2 to 100 characters"}
	}
	if query.State != "" && len([]rune(query.State)) != 2 {
		return nil, &InvoiceError{Message: "state must be a two-letter code"}
	}

	params := url.Values{}
	params.Set("name", name)
	params.Set("country", firstNonEmpty(query.Country, r.Country))
	if query.State != "" {
		params.Set("state", strings.ToUpper(query.State))
	}

	raw, err := r.send("GET", "/cities", nil, params)
	if err != nil {
		return nil, err
	}
	return decodeRows(raw)
}

func decodeRows(raw []byte) ([]map[string]any, error) {
	var parsed []map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
		}
	}
	return parsed, nil
}

// NBSSuggestion is one row of NFS-e Anexo VIII for a service code: an NBS
// with the cIndOp and cClassTrib the annex pairs with it. Description,
// OperationIndicator, IbsCbsClassification and ClassificationName are nil
// when the annex lists none, and so are the fields of the official
// cClassTrib table: ClassificationCST (for ibs_cbs.cst), ReductionIBS and
// ReductionCBS (percent reductions), LegalReference (the LC 214/2025
// article) and Incidence (where Anexo VIII says the IBS is due).
// OperationVariants holds the cIndOp for the same operation when it is not
// onerous or the acquirer is abroad, keyed onerous_domestic,
// onerous_abroad, free_domestic and free_abroad.
type NBSSuggestion struct {
	NBS                  string            `json:"nbs"`
	Description          *string           `json:"description"`
	OperationIndicator   *string           `json:"operation_indicator"`
	IbsCbsClassification *string           `json:"ibs_cbs_classification"`
	ClassificationName   *string           `json:"classification_name"`
	ClassificationCST    *string           `json:"classification_cst"`
	ReductionIBS         *float64          `json:"reduction_ibs"`
	ReductionCBS         *float64          `json:"reduction_cbs"`
	LegalReference       *string           `json:"legal_reference"`
	Incidence            *string           `json:"incidence"`
	OperationVariants    map[string]string `json:"operation_variants"`
}

// ReferenceSource is the official file the reference rows were loaded from.
type ReferenceSource struct {
	Path string  `json:"path"`
	Date *string `json:"date"`
}

// NBSSuggestions is what the NBS route answers: ServiceCode is the
// six-digit cTribNac and Suggestions every NBS, cIndOp and cClassTrib
// combination the annex lists for its subitem, empty when it lists none.
type NBSSuggestions struct {
	ServiceCode string          `json:"service_code"`
	Suggestions []NBSSuggestion `json:"suggestions"`
	Source      ReferenceSource `json:"source"`
}

// NBSSuggestions suggests the NBS codes Anexo VIII correlates with a
// service code, in any form Product.ServiceCode takes: "1.07", "01.07.00"
// or "010700". Only its LC 116 subitem decides the answer. Nothing here is
// written into a document: send a suggestion as Product.NBS and its
// OperationIndicator and IbsCbsClassification on Product.IbsCbs. A code the
// route cannot read is an *APIError 422.
func (r *FiscalReference) NBSSuggestions(serviceCode string) (*NBSSuggestions, error) {
	if len([]rune(serviceCode)) > 12 {
		return nil, &InvoiceError{Message: "service_code must be at most 12 characters"}
	}
	code, err := segment(serviceCode)
	if err != nil {
		return nil, err
	}
	raw, err := r.send("GET", "/service-codes/"+code+"/nbs", nil, nil)
	if err != nil {
		return nil, err
	}
	var parsed NBSSuggestions
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
	}
	if parsed.Suggestions == nil {
		parsed.Suggestions = []NBSSuggestion{}
	}
	return &parsed, nil
}
