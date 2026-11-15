package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

// countingInvoice returns a client whose server counts the requests that
// reach it, so a test can tell a local rejection from a round trip.
func countingInvoice(t *testing.T) (*Invoice, *int) {
	t.Helper()
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	t.Cleanup(server.Close)
	return NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key")), &hits
}

func nfeItem() br.Product {
	ncm, cfop := "84713012", "5102"
	return br.Product{Description: "Notebook", Amount: 10, NCM: &ncm, CFOP: &cfop}
}

func nfeRequest(items ...br.Product) IssueRequest {
	return IssueRequest{
		DocumentType:     NFE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            items,
		RecipientAddress: validNFEAddress(),
	}
}

// wantInvoiceError fails unless err is an *InvoiceError carrying message
// and the server was never reached.
func wantInvoiceError(t *testing.T, err error, hits int, message string) {
	t.Helper()
	var invErr *InvoiceError
	if !errors.As(err, &invErr) {
		t.Fatalf("error = %v (%T), want *InvoiceError", err, err)
	}
	if invErr.Message != message {
		t.Errorf("message = %q, want %q", invErr.Message, message)
	}
	if hits != 0 {
		t.Errorf("request reached the network %d time(s), want a local rejection", hits)
	}
}

func TestIssueRejectsAWrongGTINCheckDigit(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	barcode := "17891234567894"
	item.Barcode = &barcode

	_, err := inv.Issue(nfeRequest(item))

	wantInvoiceError(t, err, *hits, "items[0].barcode has an invalid GTIN check digit")
}
