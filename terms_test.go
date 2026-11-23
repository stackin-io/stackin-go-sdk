package stackin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

func TestTermsOmittedWhenUnset(t *testing.T) {
	raw, _, err := captureIssue(t, nfeRequest(nfeItem()))
	if err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, raw)
	for _, key := range []string{"presence", "intermediary", "transport", "payments", "change"} {
		if _, ok := body[key]; ok {
			t.Errorf("%s sent without being set", key)
		}
	}
}

func TestTermsSentWithRealZeros(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Presence = "0"
	req.Intermediary = &Intermediary{TaxID: "11222333000181", Identifier: "loja-1"}
	req.Transport = &Transport{FreightMode: "0", Volumes: []TransportVolume{{
		Quantity: ptr(0), Species: "CAIXA", NetWeight: ptr(0.0), GrossWeight: ptr(1.5),
	}}}
	req.Payments = []Payment{
		{Method: "90", Amount: 0},
		{Method: "03", Amount: 10.5, Card: &PaymentCard{Integration: "2", Brand: "01"}},
		{Method: "99", Amount: 1, Description: "Vale"},
	}
	req.Change = ptr(0.0)

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, raw)

	want := map[string]any{
		"presence":     "0",
		"intermediary": map[string]any{"tax_id": "11222333000181", "identifier": "loja-1"},
		"transport": map[string]any{"freight_mode": "0", "volumes": []any{
			map[string]any{"quantity": float64(0), "species": "CAIXA", "net_weight": float64(0), "gross_weight": 1.5},
		}},
		"payments": []any{
			map[string]any{"method": "90", "amount": "0"},
			map[string]any{"method": "03", "amount": "10.5", "card": map[string]any{"integration": "2", "brand": "01"}},
			map[string]any{"method": "99", "amount": "1", "description": "Vale"},
		},
		"change": "0",
	}
	for key, value := range want {
		if !reflect.DeepEqual(body[key], value) {
			t.Errorf("%s = %#v, want %#v", key, body[key], value)
		}
	}
}

func TestTransportVolumesNilOmittedEmptySent(t *testing.T) {
	cases := []struct {
		name string
		in   Transport
		want string
	}{
		{"nil", Transport{FreightMode: "9"}, `{"freight_mode":"9"}`},
		{"empty", Transport{FreightMode: "9", Volumes: []TransportVolume{}}, `{"freight_mode":"9","volumes":[]}`},
		{"unset volume fields", Transport{FreightMode: "1", Volumes: []TransportVolume{{}}}, `{"freight_mode":"1","volumes":[{}]}`},
	}
	for _, c := range cases {
		out, err := json.Marshal(c.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != c.want {
			t.Errorf("%s: %s, want %s", c.name, out, c.want)
		}
	}
}

func TestValidateSendsTerms(t *testing.T) {
	var raw []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"errors":[],"warnings":[]}`))
	}))
	defer server.Close()

	req := nfeRequest(nfeItem())
	req.Presence = "2"
	req.Change = ptr(0.0)
	if _, err := NewInvoice(WithBaseURL(server.URL), WithAPIKey("k")).Validate(req); err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, string(raw))
	if body["presence"] != "2" || body["change"] != "0" {
		t.Errorf("body = %v", body)
	}
}

func TestTermsLocalValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*IssueRequest)
		want   string
	}{
		{"presence", func(r *IssueRequest) { r.Presence = "7" }, "presence must be"},
		{"freight mode", func(r *IssueRequest) { r.Transport = &Transport{FreightMode: "5"} }, "freight_mode"},
		{"negative net weight", func(r *IssueRequest) {
			r.Transport = &Transport{FreightMode: "1", Volumes: []TransportVolume{{NetWeight: ptr(-1.0)}}}
		}, "net_weight can't be negative"},
		{"negative gross weight", func(r *IssueRequest) {
			r.Transport = &Transport{FreightMode: "1", Volumes: []TransportVolume{{GrossWeight: ptr(-0.1)}}}
		}, "gross_weight can't be negative"},
		{"volume net weight places", func(r *IssueRequest) {
			r.Transport = &Transport{FreightMode: "1", Volumes: []TransportVolume{{NetWeight: ptr(1.2345)}}}
		}, "net_weight allows at most 3 decimal places"},
		{"volume gross weight places", func(r *IssueRequest) {
			r.Transport = &Transport{FreightMode: "1", Volumes: []TransportVolume{{GrossWeight: ptr(0.0001)}}}
		}, "gross_weight allows at most 3 decimal places"},
		{"amount places", func(r *IssueRequest) { r.Payments = []Payment{{Method: "01", Amount: 10.005}} }, "amount allows at most 2"},
		{"negative amount", func(r *IssueRequest) { r.Payments = []Payment{{Method: "01", Amount: -1}} }, "amount can't be negative"},
		{"change places", func(r *IssueRequest) { r.Change = ptr(1.001) }, "change allows at most 2"},
		{"negative change", func(r *IssueRequest) { r.Change = ptr(-1.0) }, "change can't be negative"},
		{"method shape", func(r *IssueRequest) { r.Payments = []Payment{{Method: "1"}} }, "two-digit"},
		{"method 99 without description", func(r *IssueRequest) { r.Payments = []Payment{{Method: "99", Amount: 1}} }, "required for method 99"},
		{"empty payments", func(r *IssueRequest) { r.Payments = []Payment{} }, "payments needs 1 to 100"},
		{"too many payments", func(r *IssueRequest) { r.Payments = make([]Payment, 101) }, "payments needs 1 to 100"},
		{"identifier", func(r *IssueRequest) { r.Intermediary = &Intermediary{TaxID: "1", Identifier: "a"} }, "identifier"},
	}
	for _, c := range cases {
		req := nfeRequest(nfeItem())
		c.mutate(&req)
		expectRefusal(t, req, c.want)
	}
}

func TestTermsLeaveFiscalRulesToTheAPI(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Presence = "4"
	req.Payments = []Payment{{Method: "24", Amount: 1, Card: &PaymentCard{Integration: "1", Brand: "98"}}}
	if _, called, err := captureIssue(t, req); err != nil || !called {
		t.Errorf("err = %v called = %v, want the API to answer", err, called)
	}
}

func TestProductWeights(t *testing.T) {
	ncm, cfop := "84713012", "5102"
	data := br.Product{Description: "x", Amount: 1, NCM: &ncm, CFOP: &cfop, NetWeight: ptr(0.0), GrossWeight: ptr(0.35)}.ToDict()
	product := data["product"].(map[string]any)
	if product["net_weight"] != 0.0 || product["gross_weight"] != 0.35 {
		t.Errorf("product = %v", product)
	}
	data = br.Product{Description: "x", Amount: 1}.ToDict()
	if _, ok := data["product"].(map[string]any)["net_weight"]; ok {
		t.Error("net_weight sent when unset")
	}
}

func TestSaleTermsOfReadsTheStoredShape(t *testing.T) {
	detail := map[string]any{}
	err := json.Unmarshal([]byte(`{"result":{"id":"inv","presence":"2",
		"intermediary":{"tax_id":"11222333000181","identifier":"loja-1"},
		"transport":{"freight_mode":"1","volumes":[{"quantity":2,"species":"CAIXA","net_weight":"1.500","gross_weight":"0.000"}]},
		"payments":[{"method":"03","amount":"10.50","card":{"integration":"2","brand":"01"}},{"method":"90","amount":"0.00"}],
		"change":"0.00"}}`), &detail)
	if err != nil {
		t.Fatal(err)
	}
	terms, err := SaleTermsOf(detail)
	if err != nil {
		t.Fatal(err)
	}
	if terms.Presence == nil || *terms.Presence != "2" {
		t.Errorf("presence = %v", terms.Presence)
	}
	if terms.Intermediary.Identifier != "loja-1" {
		t.Errorf("intermediary = %+v", terms.Intermediary)
	}
	volume := terms.Transport.Volumes[0]
	if *volume.Quantity != 2 || *volume.NetWeight != 1.5 || *volume.GrossWeight != 0 {
		t.Errorf("volume = %+v", volume)
	}
	if len(terms.Payments) != 2 || terms.Payments[0].Amount != 10.5 || terms.Payments[0].Card.Brand != "01" || terms.Payments[1].Amount != 0 {
		t.Errorf("payments = %+v", terms.Payments)
	}
	if terms.Change == nil || *terms.Change != 0 {
		t.Errorf("change = %v", terms.Change)
	}
}

func TestSaleTermsOfNulls(t *testing.T) {
	terms, err := SaleTermsOf(map[string]any{"presence": nil, "intermediary": nil, "transport": nil, "payments": nil, "change": nil})
	if err != nil {
		t.Fatal(err)
	}
	if terms.Presence != nil || terms.Intermediary != nil || terms.Transport != nil || terms.Payments != nil || terms.Change != nil {
		t.Errorf("terms = %+v", terms)
	}
}

func TestItemsSendNewFields(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"id": itemID})
	if _, err := items.Create(ItemFields{
		Kind: "product", Description: "Cafe", NetWeight: ptr(0.0), GrossWeight: ptr(0.25),
		IbsCbsCST: "000", IbsCbsClass: "000001",
	}); err != nil {
		t.Fatal(err)
	}
	got := (*seen)[0].Body
	want := map[string]any{
		"kind": "product", "description": "Cafe", "net_weight": float64(0), "gross_weight": 0.25,
		"ibs_cbs_cst": "000", "ibs_cbs_class": "000001",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body %v", got)
	}
}

func TestItemsUpdateNewFields(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID})
	if _, err := items.Update(itemID, ItemChanges{
		NetWeight: ptr(0.0), IbsCbsCST: ptr("200"), IbsCbsClass: ptr(""),
	}); err != nil {
		t.Fatal(err)
	}
	got := (*seen)[0].Body
	want := map[string]any{"net_weight": float64(0), "ibs_cbs_cst": "200", "ibs_cbs_class": nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body %v", got)
	}
}

func TestItemsRefuseNegativeWeight(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{})
	if _, err := items.Create(ItemFields{Kind: "product", Description: "x", NetWeight: ptr(-1.0)}); err == nil {
		t.Error("negative weight accepted")
	}
	if _, err := items.Update(itemID, ItemChanges{GrossWeight: ptr(-1.0)}); err == nil {
		t.Error("negative weight accepted on update")
	}
	if len(*seen) != 0 {
		t.Error("request went out")
	}
}

func TestTermsAcceptAllowedPlaces(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Change = ptr(0.3)
	req.Payments = []Payment{{Method: "01", Amount: 0.1 + 0.2}, {Method: "01", Amount: 12.34}}
	req.Transport = &Transport{FreightMode: "1", Volumes: []TransportVolume{{NetWeight: ptr(1.234), GrossWeight: ptr(0.0)}}}
	if _, called, err := captureIssue(t, req); err != nil || !called {
		t.Errorf("err = %v called = %v", err, called)
	}
}

func TestCatalogWeightsAllowSixPlaces(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"id": itemID})
	if _, err := items.Create(ItemFields{Kind: "product", Description: "x", NetWeight: ptr(0.123456)}); err != nil {
		t.Fatal(err)
	}
	if len(*seen) != 1 {
		t.Error("catalog weight with 6 places was refused")
	}
	if _, err := items.Create(ItemFields{Kind: "product", Description: "x", GrossWeight: ptr(0.1234567)}); err == nil {
		t.Error("7 places accepted")
	}
	if _, err := items.Update(itemID, ItemChanges{NetWeight: ptr(0.1234567)}); err == nil {
		t.Error("7 places accepted on update")
	}
}

func TestItemsUpdateClearsWeights(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID})
	if _, err := items.Update(itemID, ItemChanges{ClearNetWeight: true, ClearGrossWeight: true}); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"net_weight": nil, "gross_weight": nil}
	if got := (*seen)[0].Body; !reflect.DeepEqual(got, want) {
		t.Errorf("body %v", got)
	}
	if _, err := items.Update(itemID, ItemChanges{ClearNetWeight: true, NetWeight: ptr(2.0)}); err != nil {
		t.Fatal(err)
	}
	if got := (*seen)[1].Body["net_weight"]; got != 2.0 {
		t.Errorf("net_weight = %v, want the set value", got)
	}
}

func TestIssueRefusesNegativeLineWeights(t *testing.T) {
	for name, product := range map[string]func(*br.Product){
		"net":   func(p *br.Product) { p.NetWeight = ptr(-0.1) },
		"gross": func(p *br.Product) { p.GrossWeight = ptr(-1.0) },
	} {
		item := nfeItem()
		product(&item)
		req := nfeRequest(item)
		expectRefusal(t, req, "can't be negative")
		_ = name
	}
}
