package stackin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func newTaxpayer(t *testing.T, status int, body any) (*Taxpayer, func() *http.Request) {
	t.Helper()
	var seen *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clone := *r
		seen = &clone
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)

	client := NewTaxpayer(WithBaseURL(server.URL), WithAPIKey("k"))
	return client, func() *http.Request { return seen }
}

func TestGetLooksUpTheExactTaxID(t *testing.T) {
	client, request := newTaxpayer(t, http.StatusOK, map[string]any{"name": "ACME"})

	if _, err := client.Get("00000000000191"); err != nil {
		t.Fatal(err)
	}

	got := request()
	if got.URL.Path != "/api/v1/taxpayers/00000000000191" {
		t.Errorf("path %q", got.URL.Path)
	}
	if got.URL.Query().Get("country") != "BR" {
		t.Errorf("country %q", got.URL.Query().Get("country"))
	}
}

func TestAnUnknownTaxIDIsAnAPIError(t *testing.T) {
	client, _ := newTaxpayer(t, http.StatusNotFound, map[string]any{"detail": "no"})

	_, err := client.Get("00000000000000")
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("got %T, want *APIError", err)
	}
}

func TestTaxpayerOneCallCanOverrideTheCountry(t *testing.T) {
	client, request := newTaxpayer(t, http.StatusOK, map[string]any{})

	if _, err := client.Get("1", "AR"); err != nil {
		t.Fatal(err)
	}

	if got := request().URL.Query().Get("country"); got != "AR" {
		t.Errorf("country %q", got)
	}
}

// A search here would be a bulk export of real people. The API refuses
// it at the route, but that is the smaller reason: this type is thin on
// purpose, and this test is what keeps a well-meaning "parity with
// FiscalReference" out.
func TestTaxpayerDeclaresExactlyOneMethodOfItsOwn(t *testing.T) {
	var own []string

	value := reflect.TypeOf(&Taxpayer{})
	embedded := reflect.TypeOf(transport{})

	for i := 0; i < value.NumMethod(); i++ {
		name := value.Method(i).Name
		if _, promoted := embedded.MethodByName(name); !promoted {
			own = append(own, name)
		}
	}

	if !reflect.DeepEqual(own, []string{"Get"}) {
		t.Errorf("public surface is %v, want [Get]", own)
	}
}

// A CNPJ is displayed with a slash; it must not rewrite the path.
func TestAFormattedCnpjStaysInsideItsSegment(t *testing.T) {
	client, request := newTaxpayer(t, http.StatusOK, map[string]any{})

	if _, err := client.Get("00.000.000/0001-91"); err != nil {
		t.Fatal(err)
	}

	if got := request().URL.EscapedPath(); got != "/api/v1/taxpayers/00.000.000%2F0001-91" {
		t.Errorf("path %q", got)
	}
}

func TestAnEmptyTaxIDIsRefusedRatherThanDropped(t *testing.T) {
	client := NewTaxpayer(WithAPIKey("k"))

	if _, err := client.Get(""); err == nil {
		t.Error("an empty tax id was accepted")
	}
}
