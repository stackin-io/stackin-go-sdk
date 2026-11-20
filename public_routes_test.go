package stackin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

type captured struct {
	method  string
	uri     string
	path    string
	query   map[string][]string
	company string
	body    map[string]any
	calls   int
}

func routeServer(t *testing.T, status int, reply string) (string, *captured) {
	t.Helper()
	got := &captured{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.calls++
		got.method = r.Method
		got.uri = r.RequestURI
		got.path = r.URL.EscapedPath()
		got.query = r.URL.Query()
		got.company = r.Header.Get("X-Stackin-Company")
		got.body = nil
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	return server.URL, got
}

func wantRefusal(t *testing.T, err error, fragment string) {
	t.Helper()
	var invErr *InvoiceError
	if !errors.As(err, &invErr) {
		t.Fatalf("err = %v, want *InvoiceError", err)
	}
	if !strings.Contains(invErr.Message, fragment) {
		t.Errorf("message = %q, want it to mention %q", invErr.Message, fragment)
	}
}

func TestStatsWithoutDaysSendsNoParams(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"total":3}`)

	result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Stats(0)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.uri != "/api/v1/invoices/stats" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	if result["total"] != float64(3) {
		t.Errorf("result = %v", result)
	}
}

func TestStatsSendsDays(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)

	if _, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).ForCompany("acme").Stats(7); err != nil {
		t.Fatal(err)
	}
	if got.query["days"][0] != "7" {
		t.Errorf("days = %v", got.query["days"])
	}
	if got.company != "acme" {
		t.Errorf("X-Stackin-Company = %q", got.company)
	}
}

func TestStatsOutsideTheAPIBoundsIsRefused(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"))

	for _, days := range []int{-1, 366} {
		_, err := inv.Stats(days)
		wantRefusal(t, err, "days")
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestReadinessReadsTheCompany(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"ready":false,"missing":["certificate"]}`)

	result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Readiness()
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.uri != "/api/v1/company" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	if result["ready"] != false {
		t.Errorf("result = %v", result)
	}
}

func TestRejectionReadsByInvoiceID(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"code":"539","steps":["a"]}`)

	result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Rejection("inv-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/invoices/inv-1/rejection" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	if result["code"] != "539" {
		t.Errorf("result = %v", result)
	}
}

func TestRejectionOfADocumentNotRejectedIsAnAPIError(t *testing.T) {
	url, _ := routeServer(t, http.StatusConflict, `{"detail":"Invoice is not rejected."}`)

	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Rejection("inv-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("err = %v, want *APIError 409", err)
	}
}

func TestInvoiceIDStaysOnePathSegment(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `[]`)
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"))

	if _, err := inv.Submissions("a/b"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/api/v1/invoices/a%2Fb/submissions" {
		t.Errorf("submissions path = %q", got.path)
	}

	url, got = routeServer(t, http.StatusOK, `{}`)
	inv = NewInvoice(WithBaseURL(url), WithAPIKey("k"))
	if _, err := inv.Rejection("a/b"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/api/v1/invoices/a%2Fb/rejection" {
		t.Errorf("rejection path = %q", got.path)
	}
}

func TestDotSegmentsAreRefused(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"))

	for _, id := range []string{"", ".", ".."} {
		_, err := inv.Rejection(id)
		wantRefusal(t, err, "path segment")
		_, err = inv.Submissions(id)
		wantRefusal(t, err, "path segment")
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestClassifySendsTheItem(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"primary_candidate":{"ncm":"22030000"}}`)

	result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Classify(ClassifyRequest{
		Description:    "Cerveja artesanal IPA 500ml",
		DocumentType:   NFE,
		RecipientState: "sp",
		Answers:        map[string]any{"purpose": "resale"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.uri != "/api/v1/ai/classify" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	want := map[string]any{
		"description":     "Cerveja artesanal IPA 500ml",
		"document_type":   "nfe",
		"recipient_state": "SP",
		"answers":         map[string]any{"purpose": "resale"},
	}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
	if result["primary_candidate"] == nil {
		t.Errorf("result = %v", result)
	}
}

func TestClassifyLeavesOutWhatWasNotGiven(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)

	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Classify(ClassifyRequest{
		Description:  "Consultoria",
		DocumentType: NFSE,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"description": "Consultoria", "document_type": "nfse"}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
}

func TestClassifyRefusesWhatTheAPIWould(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"))

	cases := []struct {
		req      ClassifyRequest
		fragment string
	}{
		{ClassifyRequest{Description: "ab", DocumentType: NFE}, "description"},
		{ClassifyRequest{Description: strings.Repeat("a", 501), DocumentType: NFE}, "description"},
		{ClassifyRequest{Description: "Cerveja"}, "document_type"},
		{ClassifyRequest{Description: "Cerveja", DocumentType: NFE, RecipientState: "SAO"}, "recipient_state"},
	}
	for _, c := range cases {
		_, err := inv.Classify(c.req)
		wantRefusal(t, err, c.fragment)
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestInvalidationsPagesTheRanges(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"items":[],"total":0}`)

	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Invalidations(PageQuery{
		Limit: 20, Offset: 40, SortBy: "created_at", OrderBy: "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/invoices/invalidations" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	want := map[string][]string{
		"limit": {"20"}, "offset": {"40"}, "sort_by": {"created_at"}, "order_by": {"asc"},
	}
	if !reflect.DeepEqual(got.query, want) {
		t.Errorf("query = %v, want %v", got.query, want)
	}
}

func TestInvalidationsWithoutAQuerySendsNone(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)

	if _, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Invalidations(PageQuery{}); err != nil {
		t.Fatal(err)
	}
	if got.uri != "/api/v1/invoices/invalidations" {
		t.Errorf("uri = %q", got.uri)
	}
}

func issueWithContingency(t *testing.T, contingency bool) map[string]any {
	t.Helper()
	url, got := routeServer(t, http.StatusOK, `{}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "123",
		Items:        []br.Product{{Description: "Widget", Amount: 10}},
		Contingency:  contingency,
	})
	if err != nil {
		t.Fatal(err)
	}
	return got.body
}

func TestContingencyIsSentWhenTrue(t *testing.T) {
	if body := issueWithContingency(t, true); body["contingency"] != true {
		t.Errorf("contingency = %v, want true", body["contingency"])
	}
}

func TestContingencyIsLeftOutWhenFalse(t *testing.T) {
	if _, ok := issueWithContingency(t, false)["contingency"]; ok {
		t.Error("contingency sent, want absent")
	}
}

func TestCitiesLooksUpByName(t *testing.T) {
	url, got := routeServer(t, http.StatusOK,
		`[{"country":"BR","code":"3544905","name":"Santa Rita do Passa Quatro","state":"SP"}]`)

	rows, err := NewFiscalReference(WithBaseURL(url), WithAPIKey("k")).
		Cities("santa rita", CityQuery{State: "sp"})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/cities" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	want := map[string][]string{"name": {"santa rita"}, "country": {"BR"}, "state": {"SP"}}
	if !reflect.DeepEqual(got.query, want) {
		t.Errorf("query = %v, want %v", got.query, want)
	}
	if len(rows) != 1 || rows[0]["code"] != "3544905" {
		t.Errorf("rows = %v", rows)
	}
}

func TestCitiesCountryFollowsTheClientUnlessGiven(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `[]`)
	ref := NewFiscalReference(WithBaseURL(url), WithAPIKey("k"), WithCountry("PT"))

	if _, err := ref.Cities("lisboa", CityQuery{}); err != nil {
		t.Fatal(err)
	}
	if got.query["country"][0] != "PT" {
		t.Errorf("country = %v", got.query["country"])
	}
	if _, ok := got.query["state"]; ok {
		t.Error("state sent, want absent")
	}

	if _, err := ref.Cities("lisboa", CityQuery{Country: "BR"}); err != nil {
		t.Fatal(err)
	}
	if got.query["country"][0] != "BR" {
		t.Errorf("country = %v", got.query["country"])
	}
}

func TestCitiesRefusesWhatTheAPIWould(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `[]`)
	ref := NewFiscalReference(WithBaseURL(url), WithAPIKey("k"))

	_, err := ref.Cities("a", CityQuery{})
	wantRefusal(t, err, "name")
	_, err = ref.Cities(strings.Repeat("a", 101), CityQuery{})
	wantRefusal(t, err, "name")
	_, err = ref.Cities("santa", CityQuery{State: "SPX"})
	wantRefusal(t, err, "state")
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}
