package stackin

import "net/url"

// Taxpayer reads the taxpayer registry, one tax id at a time.
//
// One method, and it stays one method. The registry names and
// addresses real people and companies: an exact lookup answers the
// question an issuer has — who is this CNPJ I am about to invoice —
// and answers nothing else. A prefix or wildcard search over the same
// table is a bulk export wearing the costume of a query. Do not add
// one, not for parity with FiscalReference, not "just by name", not
// "just within one state".
type Taxpayer struct {
	transport
}

func NewTaxpayer(opts ...Option) *Taxpayer {
	return &Taxpayer{transport: newTransport(opts...)}
}

// Get returns one taxpayer, by exact tax id.
//
// A 404 arrives as *APIError and does not mean the taxpayer does not
// exist: the registry reloads monthly, so a company registered in the
// last few weeks is simply not in it yet. Never build a validation
// rule on top of it.
func (t *Taxpayer) Get(taxID string, country ...string) (map[string]any, error) {
	escaped, err := segment(taxID)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("country", pickCountry(t.Country, country))

	return t.request("GET", "/taxpayers/"+escaped, nil, params)
}
