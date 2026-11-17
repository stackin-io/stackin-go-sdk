package stackin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

const clientID = "0b6c3d1e-7a2f-4e59-9c10-5d8e2f3a4b6c"

type seenRequest struct {
	Method string
	Path   string
	Query  map[string][]string
	Header http.Header
	Body   map[string]any
	Raw    string
}

func captureServer(t *testing.T, status int, reply any) (string, *[]seenRequest) {
	t.Helper()
	var seen []seenRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got := seenRequest{
			Method: r.Method,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Raw:    string(raw),
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.Body)
		}
		seen = append(seen, got)
		if status == http.StatusNoContent {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(reply)
	}))
	t.Cleanup(server.Close)

	return server.URL, &seen
}

func newClients(t *testing.T, status int, reply any, opts ...Option) (*Clients, *[]seenRequest) {
	t.Helper()
	baseURL, seen := captureServer(t, status, reply)
	return NewClients(append([]Option{WithBaseURL(baseURL), WithAPIKey("k")}, opts...)...), seen
}

func text(value string) *string {
	return &value
}

func TestClientsListSendsNoParamsWithoutAFilter(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})

	if _, err := client.List(ClientQuery{}); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodGet || got.Path != "/api/v1/clients" {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	if len(got.Query) != 0 {
		t.Errorf("query %v", got.Query)
	}
	if got.Raw != "" {
		t.Errorf("body %q", got.Raw)
	}
}

func TestClientsListSendsEveryFilterByItsAPIName(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})
	inactive := false

	_, err := client.List(ClientQuery{
		Search:   "acme",
		Source:   "issued",
		IsActive: &inactive,
		Limit:    10,
		Offset:   20,
		SortBy:   "last_used_at",
		OrderBy:  "asc",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"search":    {"acme"},
		"source":    {"issued"},
		"is_active": {"false"},
		"limit":     {"10"},
		"offset":    {"20"},
		"sort_by":   {"last_used_at"},
		"order_by":  {"asc"},
	}
	if got := (*seen)[0].Query; !reflect.DeepEqual(got, want) {
		t.Errorf("query %v, want %v", got, want)
	}
}

func TestClientsListReturnsThePage(t *testing.T) {
	page := map[string]any{"data": []any{map[string]any{"id": clientID}}, "total": float64(1)}
	client, _ := newClients(t, http.StatusOK, page)

	got, err := client.List(ClientQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, page) {
		t.Errorf("got %v", got)
	}
}

func TestClientsGetReadsOneClient(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{"id": clientID})

	if _, err := client.Get(clientID); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodGet || got.Path != "/api/v1/clients/"+clientID {
		t.Errorf("%s %s", got.Method, got.Path)
	}
}

func TestClientsGetRefusesAnIDThatLeavesItsSegment(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})

	if _, err := client.Get(".."); err == nil {
		t.Fatal("want an error")
	}
	if len(*seen) != 0 {
		t.Errorf("sent %d requests", len(*seen))
	}
}

func TestClientsCreateSendsOnlyTheFieldsSet(t *testing.T) {
	client, seen := newClients(t, http.StatusCreated, map[string]any{"id": clientID})

	if _, err := client.Create(ClientFields{Name: "ACME", TaxID: "11.222.333/0001-81"}); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodPost || got.Path != "/api/v1/clients" {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	want := map[string]any{"name": "ACME", "tax_id": "11.222.333/0001-81"}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body %v, want %v", got.Body, want)
	}
}

func TestClientsCreateSendsEveryFieldByItsAPIName(t *testing.T) {
	client, seen := newClients(t, http.StatusCreated, map[string]any{})

	_, err := client.Create(ClientFields{
		Name:                  "ACME",
		TaxID:                 "11222333000181",
		StateRegistration:     "ISENTO",
		MunicipalRegistration: "123",
		Email:                 "a@acme.test",
		Phone:                 "+5511999999999",
		Address:               &Address{State: "SP", City: "Sao Paulo"},
		Notes:                 "vip",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"name":                   "ACME",
		"tax_id":                 "11222333000181",
		"state_registration":     "ISENTO",
		"municipal_registration": "123",
		"email":                  "a@acme.test",
		"phone":                  "+5511999999999",
		"address":                map[string]any{"state": "SP", "city": "Sao Paulo"},
		"notes":                  "vip",
	}
	if got := (*seen)[0].Body; !reflect.DeepEqual(got, want) {
		t.Errorf("body %v, want %v", got, want)
	}
}

func TestClientsCreateIdentifiesABuyerAbroadByForeignID(t *testing.T) {
	client, seen := newClients(t, http.StatusCreated, map[string]any{})

	_, err := client.Create(ClientFields{
		Name:      "Globex",
		ForeignID: "P1234567",
		Address:   &Address{State: "EX", CountryCode: "2496"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"name":       "Globex",
		"foreign_id": "P1234567",
		"address":    map[string]any{"state": "EX", "country_code": "2496"},
	}
	if got := (*seen)[0].Body; !reflect.DeepEqual(got, want) {
		t.Errorf("body %v, want %v", got, want)
	}
}

func TestClientsUpdateSendsOnlyTheFieldsSet(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})

	if _, err := client.Update(clientID, ClientChanges{Email: text("new@acme.test")}); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodPatch || got.Path != "/api/v1/clients/"+clientID {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	if want := map[string]any{"email": "new@acme.test"}; !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body %v, want %v", got.Body, want)
	}
}

func TestClientsUpdateSendsNullToClear(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})

	_, err := client.Update(clientID, ClientChanges{Phone: text(""), ClearAddress: true})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{"phone": nil, "address": nil}
	if got := (*seen)[0].Body; !reflect.DeepEqual(got, want) {
		t.Errorf("body %v, want %v", got, want)
	}
}

func TestClientsDeactivateDeletes(t *testing.T) {
	client, seen := newClients(t, http.StatusNoContent, nil)

	if err := client.Deactivate(clientID); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodDelete || got.Path != "/api/v1/clients/"+clientID {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	if got.Raw != "" {
		t.Errorf("body %q", got.Raw)
	}
}

func TestClientsReactivatePatchesIsActive(t *testing.T) {
	reply := map[string]any{"id": clientID, "is_active": true}
	client, seen := newClients(t, http.StatusOK, reply)

	result, err := client.Reactivate(clientID)
	if err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodPatch || got.Path != "/api/v1/clients/"+clientID {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	if want := map[string]any{"is_active": true}; !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body %v, want %v", got.Body, want)
	}
	if !reflect.DeepEqual(result, reply) {
		t.Errorf("result %v", result)
	}
}

func TestClientsSendTheCompanyOnEveryCall(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{}, WithCompany("11222333000181"))

	_, _ = client.List(ClientQuery{})
	_, _ = client.Get(clientID)
	_, _ = client.Create(ClientFields{Name: "ACME", TaxID: "11222333000181"})
	_, _ = client.Update(clientID, ClientChanges{Name: text("ACME")})
	_ = client.Deactivate(clientID)
	_, _ = client.Reactivate(clientID)

	if len(*seen) != 6 {
		t.Fatalf("sent %d requests", len(*seen))
	}
	for _, got := range *seen {
		if company := got.Header.Get("X-Stackin-Company"); company != "11222333000181" {
			t.Errorf("%s %s company %q", got.Method, got.Path, company)
		}
	}
}

func TestClientsForCompanyBindsACopy(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{}, WithCompany("mine"))

	_, _ = client.ForCompany("other").Get(clientID)
	_, _ = client.Get(clientID)

	if got := (*seen)[0].Header.Get("X-Stackin-Company"); got != "other" {
		t.Errorf("bound company %q", got)
	}
	if got := (*seen)[1].Header.Get("X-Stackin-Company"); got != "mine" {
		t.Errorf("receiver company %q", got)
	}
}

func TestClientsSendNoCompanyHeaderWithoutOne(t *testing.T) {
	client, seen := newClients(t, http.StatusOK, map[string]any{})

	_, _ = client.List(ClientQuery{})

	if _, ok := (*seen)[0].Header["X-Stackin-Company"]; ok {
		t.Error("company header sent")
	}
}

var duplicateReply = map[string]any{
	"detail": map[string]any{
		"code":      "CLIENT_EXISTS",
		"message":   "This company already has a client with tax_id 11222333000181.",
		"client_id": clientID,
	},
}

func TestADuplicateIsAClientAlreadyExistsErrorWithTheExistingID(t *testing.T) {
	client, _ := newClients(t, http.StatusConflict, duplicateReply)

	_, err := client.Create(ClientFields{Name: "ACME", TaxID: "11222333000181"})

	var exists *ClientAlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("got %T, want *ClientAlreadyExistsError", err)
	}
	if exists.ClientID != clientID {
		t.Errorf("ClientID %q", exists.ClientID)
	}
	if exists.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode %d", exists.StatusCode)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As does not find the *APIError")
	}
}

func TestAnUpdateTakingAnotherTaxIDIsAClientAlreadyExistsError(t *testing.T) {
	client, _ := newClients(t, http.StatusConflict, duplicateReply)

	_, err := client.Update(clientID, ClientChanges{TaxID: text("11222333000181")})

	var exists *ClientAlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("got %T, want *ClientAlreadyExistsError", err)
	}
}

func TestAnotherConflictStaysAPlainAPIError(t *testing.T) {
	client, _ := newClients(t, http.StatusConflict, map[string]any{"detail": "conflict"})

	_, err := client.Create(ClientFields{Name: "ACME", TaxID: "11222333000181"})

	if _, ok := err.(*APIError); !ok {
		t.Fatalf("got %T, want *APIError", err)
	}
}

func issueSeen(t *testing.T, req IssueRequest) (map[string]any, error) {
	t.Helper()
	baseURL, seen := captureServer(t, http.StatusOK, map[string]any{})
	_, err := NewInvoice(WithBaseURL(baseURL), WithAPIKey("k")).Issue(req)
	if len(*seen) == 0 {
		return nil, err
	}
	return (*seen)[0].Body, err
}

func clientNFEItem() br.Product {
	ncm, cfop := "84716052", "5102"
	return br.Product{Description: "Teclado", Amount: 10, NCM: &ncm, CFOP: &cfop}
}

func clientNFSEItem(retained bool) br.Product {
	return br.Product{Description: "svc", Amount: 10, TaxRetained: retained}
}

func TestIssueWithClientIDAloneNamesTheRecipient(t *testing.T) {
	body, err := issueSeen(t, IssueRequest{
		DocumentType: NFE,
		ClientID:     clientID,
		Items:        []br.Product{clientNFEItem()},
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["client_id"] != clientID {
		t.Errorf("client_id %v", body["client_id"])
	}
	for _, key := range []string{"client_name", "tax_id", "recipient_address", "update_client"} {
		if _, ok := body[key]; ok {
			t.Errorf("%s sent: %v", key, body[key])
		}
	}
}

func TestIssueWithClientIDTakesTheClientsAddressOnRetainedISS(t *testing.T) {
	body, err := issueSeen(t, IssueRequest{
		DocumentType: NFSE,
		ClientID:     clientID,
		Items:        []br.Product{clientNFSEItem(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["recipient_address"]; ok {
		t.Error("recipient_address sent")
	}
}

func TestIssueWithClientIDSendsTheFieldsSetSoTheyWin(t *testing.T) {
	body, err := issueSeen(t, IssueRequest{
		DocumentType: NFSE,
		ClientID:     clientID,
		ClientName:   "ACME Filial",
		TaxID:        "11222333000181",
		Items:        []br.Product{clientNFSEItem(false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if body["client_name"] != "ACME Filial" || body["tax_id"] != "11222333000181" {
		t.Errorf("body %v", body)
	}
}

func TestIssueWithClientIDStillRefusesAPartialAddress(t *testing.T) {
	_, err := issueSeen(t, IssueRequest{
		DocumentType:     NFE,
		ClientID:         clientID,
		Items:            []br.Product{clientNFEItem()},
		RecipientAddress: &Address{State: "SP"},
	})
	if err == nil {
		t.Fatal("want an error")
	}
}

func TestIssueSendsUpdateClientOnlyWhenTrue(t *testing.T) {
	body, err := issueSeen(t, IssueRequest{
		DocumentType: NFE,
		ClientID:     clientID,
		Items:        []br.Product{clientNFEItem()},
		UpdateClient: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if body["update_client"] != true {
		t.Errorf("update_client %v", body["update_client"])
	}
}

func TestIssueWithoutClientIDStillRequiresTheNFEAddress(t *testing.T) {
	_, err := issueSeen(t, IssueRequest{
		DocumentType: NFE,
		ClientName:   "ACME",
		TaxID:        "11222333000181",
		Items:        []br.Product{clientNFEItem()},
	})
	if err == nil {
		t.Fatal("want an error")
	}
}

func TestIssueWithoutClientIDSendsTheSameBody(t *testing.T) {
	body, err := issueSeen(t, IssueRequest{
		DocumentType: NFSE,
		ClientName:   "ACME",
		TaxID:        "11222333000181",
		Items:        []br.Product{clientNFSEItem(false)},
	})
	if err != nil {
		t.Fatal(err)
	}

	keys := make([]string, 0, len(body))
	for key := range body {
		keys = append(keys, key)
	}
	want := map[string]bool{"document_type": true, "client_name": true, "tax_id": true, "items": true}
	if len(keys) != len(want) {
		t.Fatalf("keys %v", keys)
	}
	for _, key := range keys {
		if !want[key] {
			t.Errorf("unexpected key %s", key)
		}
	}
}
