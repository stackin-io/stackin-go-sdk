package stackin

import (
	"encoding/json"
	"net/url"
	"strconv"
)

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
	params := url.Values{}
	params.Set("country", pickCountry(k.Country, country))

	return k.client.request("GET", "/fiscal-references/"+k.Name+"/"+code, nil, params)
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
