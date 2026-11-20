package stackin

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const webhookSecret = "0123456789abcdef"

func TestWebhookCreateSendsTheEndpoint(t *testing.T) {
	url, got := routeServer(t, http.StatusCreated, `{"id":"wh-1","url":"https://shop.example/stackin"}`)

	endpoint, err := NewWebhooks(WithBaseURL(url), WithAPIKey("k")).Create(WebhookEndpoint{
		URL:    "https://shop.example/stackin",
		Secret: webhookSecret,
		Events: []string{"invoice.authorized", "invoice.rejected"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.uri != "/api/v1/webhooks" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	want := map[string]any{
		"url":    "https://shop.example/stackin",
		"secret": webhookSecret,
		"events": []any{"invoice.authorized", "invoice.rejected"},
	}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
	if endpoint["id"] != "wh-1" {
		t.Errorf("endpoint = %v", endpoint)
	}
}

func TestWebhookCreateWithoutEventsLeavesThemOut(t *testing.T) {
	url, got := routeServer(t, http.StatusCreated, `{}`)

	_, err := NewWebhooks(WithBaseURL(url), WithAPIKey("k")).Create(WebhookEndpoint{
		URL: "HTTPS://shop.example/stackin", Secret: webhookSecret,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.body["events"]; ok {
		t.Error("events sent, want absent")
	}
}

func TestWebhookCreateRefusesWhatTheAPIWould(t *testing.T) {
	url, got := routeServer(t, http.StatusCreated, `{}`)
	hooks := NewWebhooks(WithBaseURL(url), WithAPIKey("k"))

	cases := []struct {
		endpoint WebhookEndpoint
		fragment string
	}{
		{WebhookEndpoint{URL: "http://shop.example", Secret: webhookSecret}, "https://"},
		{WebhookEndpoint{URL: "https://" + strings.Repeat("a", 500), Secret: webhookSecret}, "at most 500"},
		{WebhookEndpoint{URL: "https://shop.example", Secret: "short"}, "secret"},
		{WebhookEndpoint{URL: "https://shop.example", Secret: strings.Repeat("s", 256)}, "secret"},
		{WebhookEndpoint{
			URL: "https://shop.example", Secret: webhookSecret,
			Events: []string{"invoice.paid", "invoice.authorized"},
		}, "unknown events: invoice.paid. Known: invoice.authorized"},
	}
	for _, c := range cases {
		_, err := hooks.Create(c.endpoint)
		wantRefusal(t, err, c.fragment)
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestWebhookListIsAPlainList(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `[{"id":"wh-1"},{"id":"wh-2"}]`)

	rows, err := NewWebhooks(WithBaseURL(url), WithAPIKey("k")).ForCompany("acme").List()
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.uri != "/api/v1/webhooks" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	if got.company != "acme" {
		t.Errorf("X-Stackin-Company = %q", got.company)
	}
	if len(rows) != 2 || rows[1]["id"] != "wh-2" {
		t.Errorf("rows = %v", rows)
	}
}

func TestWebhookDeleteEscapesTheID(t *testing.T) {
	url, got := routeServer(t, http.StatusNoContent, ``)
	hooks := NewWebhooks(WithBaseURL(url), WithAPIKey("k"))

	if err := hooks.Delete("a/b"); err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodDelete || got.path != "/api/v1/webhooks/a%2Fb" {
		t.Errorf("request = %s %s", got.method, got.path)
	}

	wantRefusal(t, hooks.Delete(".."), "path segment")
	if got.calls != 1 {
		t.Errorf("calls = %d, want 1", got.calls)
	}
}

func TestWebhookDeleteOfAnUnknownIDIsAnAPIError(t *testing.T) {
	url, _ := routeServer(t, http.StatusNotFound, `{"detail":"webhook not found"}`)

	err := NewWebhooks(WithBaseURL(url), WithAPIKey("k")).Delete("wh-9")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want *APIError 404", err)
	}
}

func TestWebhookDeliveriesPages(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"items":[{"event":"invoice.authorized"}],"total":1}`)

	page, err := NewWebhooks(WithBaseURL(url), WithAPIKey("k")).Deliveries(PageQuery{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/webhooks/deliveries" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	if !reflect.DeepEqual(got.query, map[string][]string{"limit": {"20"}}) {
		t.Errorf("query = %v", got.query)
	}
	if page["total"] != float64(1) {
		t.Errorf("page = %v", page)
	}
}

func TestWebhookForCompanyLeavesTheReceiverAlone(t *testing.T) {
	hooks := NewWebhooks(WithAPIKey("k"), WithCompany("one"))
	bound := hooks.ForCompany("two")

	if hooks.Company != "one" || bound.Company != "two" {
		t.Errorf("receiver %q, bound %q", hooks.Company, bound.Company)
	}
}
