package stackin

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// WebhookEvents are the event names an endpoint can subscribe to.
var WebhookEvents = []string{
	"invoice.authorized",
	"invoice.rejected",
	"invoice.cancelled",
	"invoice.corrected",
	"received_invoice.stored",
	"received_invoice.full_document",
}

const (
	webhookSecretMin = 16
	webhookSecretMax = 255
	webhookURLMax    = 500
)

// Webhooks manages where the company's events are delivered (/webhooks).
//
// It is company data: WithCompany and ForCompany pick the company. An
// endpoint receives the events of the credential's own environment only.
// On an OAuth credential every call needs webhook:manage; an API key has
// it.
type Webhooks struct {
	transport
}

func NewWebhooks(opts ...Option) *Webhooks {
	return &Webhooks{transport: newTransport(opts...)}
}

// ForCompany returns a copy of the client bound to another company, for
// the calls chained on it. The receiver is left as it was.
func (w *Webhooks) ForCompany(company string) *Webhooks {
	bound := *w
	bound.Company = company
	return &bound
}

// WebhookEndpoint is an endpoint as Create takes it.
type WebhookEndpoint struct {
	// URL must be https://, at most 500 characters. The API also refuses
	// one that points at a private network.
	URL string
	// Secret signs every delivery, 16 to 255 characters. It is never
	// returned again, so keep it.
	Secret string
	// Events narrows what is sent, from WebhookEvents. Left nil, the
	// endpoint receives every event.
	Events []string
}

// Create registers an HTTPS endpoint for this company's events.
func (w *Webhooks) Create(endpoint WebhookEndpoint) (map[string]any, error) {
	if !strings.HasPrefix(strings.ToLower(endpoint.URL), "https://") {
		return nil, &InvoiceError{Message: "url must be an https:// URL"}
	}
	if len([]rune(endpoint.URL)) > webhookURLMax {
		return nil, &InvoiceError{Message: fmt.Sprintf("url must be at most %d characters", webhookURLMax)}
	}
	length := len([]rune(endpoint.Secret))
	if length < webhookSecretMin || length > webhookSecretMax {
		return nil, &InvoiceError{Message: fmt.Sprintf(
			"secret must be %d to %d characters", webhookSecretMin, webhookSecretMax,
		)}
	}

	payload := map[string]any{"url": endpoint.URL, "secret": endpoint.Secret}
	if endpoint.Events != nil {
		if unknown := unknownEvents(endpoint.Events); len(unknown) > 0 {
			return nil, &InvoiceError{Message: fmt.Sprintf(
				"unknown events: %s. Known: %s",
				strings.Join(unknown, ", "), strings.Join(WebhookEvents, ", "),
			)}
		}
		payload["events"] = endpoint.Events
	}
	return w.request(http.MethodPost, "/webhooks", payload, nil)
}

func unknownEvents(events []string) []string {
	known := map[string]bool{}
	for _, event := range WebhookEvents {
		known[event] = true
	}
	seen := map[string]bool{}
	var unknown []string
	for _, event := range events {
		if !known[event] && !seen[event] {
			seen[event] = true
			unknown = append(unknown, event)
		}
	}
	sort.Strings(unknown)
	return unknown
}

// List returns this company's endpoints in the credential's environment.
// A plain list, not a page. Secrets are never returned.
func (w *Webhooks) List() ([]map[string]any, error) {
	raw, err := w.send(http.MethodGet, "/webhooks", nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeRows(raw)
}

// Delete stops delivery to one endpoint, keeping what was already sent.
// An id this company does not have is an *APIError 404.
func (w *Webhooks) Delete(endpointID string) error {
	id, err := segment(endpointID)
	if err != nil {
		return err
	}
	_, err = w.send(http.MethodDelete, "/webhooks/"+id, nil, nil)
	return err
}

// Deliveries lists every delivery attempt in the credential's
// environment, a page. Each row has the event, its status, attempts, the
// endpoint's response_status, last_error and when it was delivered or will
// be tried again.
func (w *Webhooks) Deliveries(query PageQuery) (map[string]any, error) {
	return w.request(http.MethodGet, "/webhooks/deliveries", nil, query.values())
}
