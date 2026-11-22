package stackin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackin-io/stackin-go-sdk/br"
)

// captureIssue issues req against a test server and returns the raw body
// sent, or the error Issue returned before or after the call.
func captureIssue(t *testing.T, req IssueRequest) (string, bool, error) {
	t.Helper()
	var raw []byte
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		raw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()

	_, err := NewInvoice(WithBaseURL(server.URL), WithAPIKey("k")).Issue(req)
	return string(raw), called, err
}

func sentFields(t *testing.T, raw string) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", raw, err)
	}
	return body
}

func abroadAddress() *Address {
	return &Address{
		Street:       "Fifth Avenue",
		Number:       "350",
		Neighborhood: "Midtown",
		City:         "New York",
		State:        "EX",
		CountryCode:  "2496",
		CountryName:  "Estados Unidos",
	}
}

func expectRefusal(t *testing.T, req IssueRequest, want string) {
	t.Helper()
	_, called, err := captureIssue(t, req)
	var invErr *InvoiceError
	if !errors.As(err, &invErr) {
		t.Fatalf("err = %v, want an *InvoiceError", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("err = %q, want it to contain %q", err.Error(), want)
	}
	if called {
		t.Error("request went out, want a local refusal")
	}
}

func TestIssueWithoutNewFieldsSendsTheSameBody(t *testing.T) {
	raw, _, err := captureIssue(t, nfeRequest(nfeItem()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `{"client_name":"Acme","document_type":"nfe","items":[{"amount":"10","description":"Notebook",` +
		`"product":{"br":{"cfop":"5102","ncm":"84713012"},"quantity":1,"unit":"UN","used_movable_asset":false},` +
		`"tax_retained":false}],"recipient_address":{"state":"SC","city_code":"4209102","street":"Rua das Flores",` +
		`"number":"1200","neighborhood":"Centro","city":"Joinville","zip_code":"89201100"},"tax_id":"11222333000181"}`
	if raw != want {
		t.Errorf("body =\n%s\nwant\n%s", raw, want)
	}
}

func TestIssueSendsDiscountAsDecimal(t *testing.T) {
	req := nfeRequest(nfeItem())
	discount := 12.5
	req.Discount = &discount

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sentFields(t, raw)["discount"]; got != "12.5" {
		t.Errorf("discount = %#v, want \"12.5\"", got)
	}
}

func TestIssueRefusesNegativeDiscount(t *testing.T) {
	req := nfeRequest(nfeItem())
	discount := -1.0
	req.Discount = &discount
	expectRefusal(t, req, "discount must be zero or more")
}

func TestIssueSendsIssuedAtWithItsOwnOffset(t *testing.T) {
	req := nfeRequest(nfeItem())
	issuedAt := time.Date(2026, 11, 14, 9, 30, 0, 0, time.FixedZone("BRT", -3*60*60))
	req.IssuedAt = &issuedAt

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sentFields(t, raw)["issued_at"]; got != "2026-11-14T09:30:00-03:00" {
		t.Errorf("issued_at = %#v, want 2026-11-14T09:30:00-03:00", got)
	}
}

func TestIssueSendsUTCIssuedAtWithZ(t *testing.T) {
	req := nfeRequest(nfeItem())
	issuedAt := time.Date(2026, 11, 14, 12, 30, 0, 0, time.UTC)
	req.IssuedAt = &issuedAt

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sentFields(t, raw)["issued_at"]; got != "2026-11-14T12:30:00Z" {
		t.Errorf("issued_at = %#v, want 2026-11-14T12:30:00Z", got)
	}
}

func TestIssueRefusesZeroIssuedAt(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.IssuedAt = &time.Time{}
	expectRefusal(t, req, "issued_at")
}

func TestIssueSendsPurposeDirectionAndReferencedKeys(t *testing.T) {
	req := nfeRequest(nfeItem())
	key := "35261111222333000181550010000000011000000010"
	req.Purpose = PurposeReturn
	req.Direction = DirectionIn
	req.ReferencedKeys = []string{key}

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := sentFields(t, raw)
	if body["purpose"] != "return" {
		t.Errorf("purpose = %#v, want return", body["purpose"])
	}
	if body["direction"] != "in" {
		t.Errorf("direction = %#v, want in", body["direction"])
	}
	keys, _ := body["referenced_keys"].([]any)
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("referenced_keys = %#v, want [%s]", body["referenced_keys"], key)
	}
}

func TestIssueOmitsEmptyReferencedKeys(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.ReferencedKeys = []string{}

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, present := sentFields(t, raw)["referenced_keys"]; present {
		t.Error("referenced_keys sent, want it left out when empty")
	}
}

func TestIssueRefusesUnknownPurpose(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Purpose = "devolucao"
	expectRefusal(t, req, "purpose must be")
}

func TestIssueRefusesUnknownDirection(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Direction = "inbound"
	expectRefusal(t, req, "direction must be")
}

func TestIssueToBuyerAbroadLeavesTaxIDOut(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.TaxID = ""
	req.RecipientAddress = abroadAddress()
	req.ForeignID = "P1234567"
	req.Export = &Export{ExitState: "SP", Place: "Porto de Santos"}

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := sentFields(t, raw)
	if _, present := body["tax_id"]; present {
		t.Errorf("tax_id = %#v, want it left out for a buyer abroad", body["tax_id"])
	}
	if body["foreign_id"] != "P1234567" {
		t.Errorf("foreign_id = %#v, want P1234567", body["foreign_id"])
	}
	if !strings.Contains(raw, `"export":{"exit_state":"SP","place":"Porto de Santos"}`) {
		t.Errorf("body = %s, want export without dispatch_place", raw)
	}
	address := body["recipient_address"].(map[string]any)
	if address["country_code"] != "2496" || address["country_name"] != "Estados Unidos" {
		t.Errorf("recipient_address = %#v, want the country sent", address)
	}
	if _, present := address["zip_code"]; present {
		t.Errorf("recipient_address = %#v, want zip_code left out", address)
	}
}

func TestIssueSendsExportDispatchPlace(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.TaxID = ""
	req.RecipientAddress = abroadAddress()
	req.Export = &Export{ExitState: "PR", Place: "Paranagua", DispatchPlace: "Recinto 1"}

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(raw, `"export":{"exit_state":"PR","place":"Paranagua","dispatch_place":"Recinto 1"}`) {
		t.Errorf("body = %s, want the full export group", raw)
	}
}

func TestIssueKeepsEmptyTaxIDInBrazil(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.TaxID = ""

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, present := sentFields(t, raw)["tax_id"]; !present || got != "" {
		t.Errorf("tax_id = %#v, want it sent as before", got)
	}
}

func TestIssueAbroadNeedsCountryCode(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.RecipientAddress = abroadAddress()
	req.RecipientAddress.CountryCode = ""
	expectRefusal(t, req, "CountryCode")
}

func TestIssueAbroadAcceptsLowercaseState(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.TaxID = ""
	req.RecipientAddress = abroadAddress()
	req.RecipientAddress.State = "ex"

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, present := sentFields(t, raw)["tax_id"]; present {
		t.Error("tax_id sent, want it left out for state ex")
	}
}

func TestIssueInBrazilStillNeedsZipCode(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.RecipientAddress.ZipCode = ""
	expectRefusal(t, req, "ZipCode")
}

func TestIssueDoesNotCheckTheBACENCheckDigit(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.RecipientAddress = abroadAddress()
	req.RecipientAddress.CountryCode = "1"

	if _, _, err := captureIssue(t, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNFSeWithRetainedISSStillNeedsZipCodeForEX(t *testing.T) {
	req := IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Consulting", Amount: 10, TaxRetained: ptr(true)}},
		RecipientAddress: abroadAddress(),
	}
	expectRefusal(t, req, "ZipCode")
}

func TestIssueRefusesNonLatin1InNewTextFields(t *testing.T) {
	cases := map[string]func(*IssueRequest){
		"recipient_address.country_name": func(r *IssueRequest) {
			r.RecipientAddress = abroadAddress()
			r.RecipientAddress.CountryName = "日本"
		},
		"export.place": func(r *IssueRequest) {
			r.Export = &Export{ExitState: "SP", Place: "Porto 🚢"}
		},
		"items[0].product.br.fuel.anp_description": func(r *IssueRequest) {
			r.Items[0].Fuel = &br.Fuel{AnpCode: "320102001", AnpDescription: "Diesel ✓", ConsumerState: "SP"}
		},
	}
	for path, mutate := range cases {
		t.Run(path, func(t *testing.T) {
			req := nfeRequest(nfeItem())
			mutate(&req)
			expectRefusal(t, req, path)
		})
	}
}

func TestLatin1CheckIgnoresIssuedAtAndReferencedKeys(t *testing.T) {
	req := nfeRequest(nfeItem())
	issuedAt := time.Date(2026, 11, 14, 9, 30, 0, 0, time.FixedZone("東京", 9*60*60))
	req.IssuedAt = &issuedAt
	req.ReferencedKeys = []string{"35261111222333000181550010000000011000000010"}

	if _, _, err := captureIssue(t, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueSendsFuelUnderBr(t *testing.T) {
	req := nfeRequest(nfeItem())
	biodiesel := 14.0
	req.Items[0].Fuel = &br.Fuel{
		AnpCode:          "820101034",
		AnpDescription:   "OLEO DIESEL B S10",
		ConsumerState:    "SP",
		BiodieselPercent: &biodiesel,
	}

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `"fuel":{"anp_code":"820101034","anp_description":"OLEO DIESEL B S10","biodiesel_percent":14,"consumer_state":"SP"}`
	if !strings.Contains(raw, want) {
		t.Errorf("body = %s, want %s", raw, want)
	}
}

func TestIssueRefusesFuelWithoutRequiredFields(t *testing.T) {
	for _, field := range []string{"anp_code", "anp_description", "consumer_state"} {
		t.Run(field, func(t *testing.T) {
			fuel := br.Fuel{AnpCode: "820101034", AnpDescription: "OLEO DIESEL", ConsumerState: "SP"}
			switch field {
			case "anp_code":
				fuel.AnpCode = ""
			case "anp_description":
				fuel.AnpDescription = ""
			case "consumer_state":
				fuel.ConsumerState = ""
			}
			req := nfeRequest(nfeItem())
			req.Items[0].Fuel = &fuel
			expectRefusal(t, req, "items[0].fuel."+field+" is required")
		})
	}
}

func TestIssueRefusesFuelBesideExtraGroupsComb(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Items[0].Fuel = &br.Fuel{AnpCode: "820101034", AnpDescription: "OLEO DIESEL", ConsumerState: "SP"}
	req.Items[0].ExtraGroups = map[string]any{"comb": map[string]any{"cProdANP": "820101034"}}
	expectRefusal(t, req, "items[0].fuel can't be sent together with extra_groups.comb")
}
