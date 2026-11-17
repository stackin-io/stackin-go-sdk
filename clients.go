package stackin

import (
	"net/http"
	"net/url"
	"strconv"
)

// Clients reads and edits the company's saved clients: the recipients
// it issues to again. A successful issuance saves a new recipient on its
// own, and IssueRequest.ClientID issues to a saved one.
//
// It is company data, like everything on Invoice: WithCompany and
// ForCompany pick the company. Reading needs invoice:read; creating,
// editing and deactivating need invoice:issue. The rules on a client's
// fields are the issuance's own and they are the API's: a refusal comes
// back as an *APIError 422.
type Clients struct {
	transport
}

func NewClients(opts ...Option) *Clients {
	return &Clients{transport: newTransport(opts...)}
}

// ForCompany returns a copy of the client bound to another company, for
// the calls chained on it. The receiver is left as it was.
func (c *Clients) ForCompany(company string) *Clients {
	bound := *c
	bound.Company = company
	return &bound
}

// ClientQuery narrows the client listing. Every field is optional; the
// zero value asks for the first page of active clients, by name.
type ClientQuery struct {
	// Search matches part of the name, the tax id (masked or not) or the
	// foreign id.
	Search string
	// Source is manual, issued or issuance.
	Source string
	// IsActive left nil lists the active clients, as the API does;
	// pointing at false lists the deactivated ones.
	IsActive *bool
	Limit    int
	Offset   int
	// SortBy takes name, last_used_at, created_at or uses_count.
	SortBy  string
	OrderBy string
}

// ClientFields is a client as Create takes it. TaxID is required except
// for a buyer abroad (Address.State EX), who has a ForeignID instead.
// Empty fields are left out of the request.
type ClientFields struct {
	Name                  string   `json:"name"`
	TaxID                 string   `json:"tax_id,omitempty"`
	ForeignID             string   `json:"foreign_id,omitempty"`
	StateRegistration     string   `json:"state_registration,omitempty"`
	MunicipalRegistration string   `json:"municipal_registration,omitempty"`
	Email                 string   `json:"email,omitempty"`
	Phone                 string   `json:"phone,omitempty"`
	Address               *Address `json:"address,omitempty"`
	Notes                 string   `json:"notes,omitempty"`
}

// ClientChanges is what Update changes. A nil field is not sent; a
// field pointing at "" is sent as null, which clears it. Address
// replaces the whole address; ClearAddress removes it.
type ClientChanges struct {
	Name                  *string
	TaxID                 *string
	ForeignID             *string
	StateRegistration     *string
	MunicipalRegistration *string
	Email                 *string
	Phone                 *string
	Address               *Address
	ClearAddress          bool
	Notes                 *string
	// IsActive pointing at true brings a deactivated client back; see
	// Reactivate.
	IsActive *bool
}

func (changes ClientChanges) body() map[string]any {
	body := map[string]any{}
	texts := []struct {
		key   string
		value *string
	}{
		{"name", changes.Name},
		{"tax_id", changes.TaxID},
		{"foreign_id", changes.ForeignID},
		{"state_registration", changes.StateRegistration},
		{"municipal_registration", changes.MunicipalRegistration},
		{"email", changes.Email},
		{"phone", changes.Phone},
		{"notes", changes.Notes},
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
	if changes.Address != nil {
		body["address"] = changes.Address
	} else if changes.ClearAddress {
		body["address"] = nil
	}
	if changes.IsActive != nil {
		body["is_active"] = *changes.IsActive
	}
	return body
}

// List returns a page of the company's clients.
func (c *Clients) List(query ClientQuery) (map[string]any, error) {
	params := url.Values{}
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
	return c.request(http.MethodGet, "/clients", nil, params)
}

// Get returns one client, deactivated or not. Another company's client
// is a 404.
func (c *Clients) Get(clientID string) (map[string]any, error) {
	escaped, err := segment(clientID)
	if err != nil {
		return nil, err
	}
	return c.request(http.MethodGet, "/clients/"+escaped, nil, nil)
}

// Create saves a client by hand. A TaxID or ForeignID the company
// already holds is a *ClientAlreadyExistsError, whose ClientID names it.
func (c *Clients) Create(fields ClientFields) (map[string]any, error) {
	return c.request(http.MethodPost, "/clients", fields, nil)
}

// Update changes only the fields set in changes. The API checks the
// client as a whole, so clearing a Brazilian client's tax id is refused
// like a create without one; taking another client's tax id is a
// *ClientAlreadyExistsError.
func (c *Clients) Update(clientID string, changes ClientChanges) (map[string]any, error) {
	escaped, err := segment(clientID)
	if err != nil {
		return nil, err
	}
	return c.request(http.MethodPatch, "/clients/"+escaped, changes.body(), nil)
}

// Deactivate deactivates a client. It is kept, left out of List by
// default and refused on issue until Reactivate brings it back.
func (c *Clients) Deactivate(clientID string) error {
	escaped, err := segment(clientID)
	if err != nil {
		return err
	}
	_, err = c.send(http.MethodDelete, "/clients/"+escaped, nil, nil)
	return err
}

// Reactivate brings a deactivated client back, and returns it.
func (c *Clients) Reactivate(clientID string) (map[string]any, error) {
	active := true
	return c.Update(clientID, ClientChanges{IsActive: &active})
}
