package stackin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newReference(t *testing.T, body any) (*FiscalReference, func() *http.Request) {
	t.Helper()
	var seen *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clone := *r
		seen = &clone
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)

	client := NewFiscalReference(WithBaseURL(server.URL), WithAPIKey("k"))
	return client, func() *http.Request { return seen }
}

func TestEveryDocumentedKindHasAnAccessor(t *testing.T) {
	client := NewFiscalReference(WithAPIKey("k"))

	accessors := map[string]*Kind{
		"cfop":          client.CFOP,
		"ncm":           client.NCM,
		"cest":          client.CEST,
		"cst":           client.CST,
		"csosn":         client.CSOSN,
		"iss_service":   client.IssService,
		"icms_fuel":     client.IcmsFuel,
		"ibs_cbs_class": client.IbsCbsClass,
	}

	if len(accessors) != len(Kinds) {
		t.Fatalf("Kinds has %d entries, accessors %d", len(Kinds), len(accessors))
	}
	for _, name := range Kinds {
		kind, ok := accessors[name]
		if !ok {
			t.Fatalf("%s is in Kinds with no accessor", name)
		}
		if kind.Name != name {
			t.Errorf("accessor for %s is bound to %s", name, kind.Name)
		}
	}
}

func TestAKindWithNoAccessorIsStillReachable(t *testing.T) {
	client := NewFiscalReference(WithAPIKey("k"))

	if got := client.Kind("published_tomorrow").Name; got != "published_tomorrow" {
		t.Errorf("got %q", got)
	}
}

func TestGetAsksForTheOneCode(t *testing.T) {
	client, request := newReference(t, map[string]any{"code": "84716052"})

	if _, err := client.NCM.Get("84716052"); err != nil {
		t.Fatal(err)
	}

	got := request()
	if got.URL.Path != "/api/v1/fiscal-references/ncm/84716052" {
		t.Errorf("path %q", got.URL.Path)
	}
	if got.URL.Query().Get("country") != "BR" {
		t.Errorf("country %q", got.URL.Query().Get("country"))
	}
}

func TestAMissingCodeIsAnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"no"}`))
	}))
	defer server.Close()

	client := NewFiscalReference(WithBaseURL(server.URL), WithAPIKey("k"))

	_, err := client.CFOP.Get("9999")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("status %d", apiErr.StatusCode)
	}
}

func TestSearchPinsTheKindItWasReachedThrough(t *testing.T) {
	client, request := newReference(t, map[string]any{"data": []any{}})

	if _, err := client.NCM.Search(SearchQuery{Term: "teclado"}); err != nil {
		t.Fatal(err)
	}

	query := request().URL.Query()
	if query.Get("kind") != "ncm" {
		t.Errorf("kind %q", query.Get("kind"))
	}
	if query.Get("search") != "teclado" {
		t.Errorf("search %q", query.Get("search"))
	}
}

func TestAnEmptyTermIsNotSentAsAnEmptySearch(t *testing.T) {
	client, request := newReference(t, map[string]any{"data": []any{}})

	if _, err := client.CFOP.Search(SearchQuery{Term: ""}); err != nil {
		t.Fatal(err)
	}

	if _, present := request().URL.Query()["search"]; present {
		t.Error("sent search= for an empty term; the route 422s on it")
	}
}

func TestSearchingTheClientItselfPinsNoKind(t *testing.T) {
	client, request := newReference(t, map[string]any{"data": []any{}})

	if _, err := client.Search(SearchQuery{Term: "teclado"}); err != nil {
		t.Fatal(err)
	}

	if _, present := request().URL.Query()["kind"]; present {
		t.Error("pinned a kind on a search meant to span every kind")
	}
}

func TestSearchSendsNoOrderingTheRouteWouldDiscard(t *testing.T) {
	client, request := newReference(t, map[string]any{"data": []any{}})

	if _, err := client.NCM.Search(SearchQuery{Limit: 10, Offset: 20}); err != nil {
		t.Fatal(err)
	}

	query := request().URL.Query()
	if query.Get("limit") != "10" || query.Get("offset") != "20" {
		t.Errorf("limit %q offset %q", query.Get("limit"), query.Get("offset"))
	}
	for _, unwanted := range []string{"sort_by", "order_by"} {
		if _, present := query[unwanted]; present {
			t.Errorf("sent %s, which the route discards", unwanted)
		}
	}
}

func TestCountryDefaultsToBRAndReachesEveryAccessor(t *testing.T) {
	if got := NewFiscalReference(WithAPIKey("k")).Country; got != "BR" {
		t.Fatalf("default country %q", got)
	}

	client := NewFiscalReference(WithAPIKey("k"), WithCountry("AR"))
	if got := client.NCM.Country; got != "AR" {
		t.Errorf("accessor country %q", got)
	}
}

func TestOneCallCanOverrideTheCountry(t *testing.T) {
	client, request := newReference(t, map[string]any{})

	if _, err := client.NCM.Get("1", "PY"); err != nil {
		t.Fatal(err)
	}

	if got := request().URL.Query().Get("country"); got != "PY" {
		t.Errorf("country %q", got)
	}
}

func TestKindsAsksTheAPIRatherThanAnsweringFromTheSlice(t *testing.T) {
	client, request := newReference(t, []string{"ncm", "brand_new"})

	got, err := client.Kinds()
	if err != nil {
		t.Fatal(err)
	}

	if request().URL.Path != "/api/v1/fiscal-references/kinds" {
		t.Errorf("path %q", request().URL.Path)
	}
	if len(got) != 2 || got[1] != "brand_new" {
		t.Errorf("got %v", got)
	}
}
