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

func TestIssueRejectsNonLatin1TextOnNFE(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*IssueRequest)
		message string
	}{
		{
			"emoji in description",
			func(r *IssueRequest) { r.Items[0].Description = "Notebook 💻" },
			"items[0].description contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '💻'",
		},
		{
			"CJK in client name",
			func(r *IssueRequest) { r.ClientName = "Acme 中 Ltda" },
			"client_name contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '中'",
		},
		{
			"Cyrillic in street",
			func(r *IssueRequest) { r.RecipientAddress.Street = "Rua Ж" },
			"recipient_address.street contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: 'Ж'",
		},
		{
			"emoji in purchase order",
			func(r *IssueRequest) { po := "PO-🔥"; r.Items[0].PurchaseOrder = &po },
			"items[0].product.purchase_order contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '🔥'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv, hits := countingInvoice(t)
			req := nfeRequest(nfeItem())
			tc.mutate(&req)

			_, err := inv.Issue(req)

			wantInvoiceError(t, err, *hits, tc.message)
		})
	}
}

func TestIssueAcceptsLatin1AccentsOnNFE(t *testing.T) {
	inv, hits := countingInvoice(t)
	req := nfeRequest(nfeItem())
	req.ClientName = "Pão de Açúcar Comércio Ltda"
	req.Items[0].Description = "Café torrado ÿ ½ ©"
	req.RecipientAddress.Neighborhood = "São João"

	if _, err := inv.Issue(req); err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueLeavesWhatTheAPIConvertsOnNFE(t *testing.T) {
	inv, hits := countingInvoice(t)
	req := nfeRequest(nfeItem())
	req.Items[0].Description = "“Teclado” — 2 un… €10™"
	req.Items[0].Unit = "a\tb\nc"
	obs := "‘a’ ‚b „c” – − • é ç ã\r\n"
	req.Items[0].Observations = &obs
	req.ClientName = "é ç ã"

	if _, err := inv.Issue(req); err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueDoesNotApplyLatin1ToNFSE(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme “Ltda”",
		TaxID:        "11222333000181",
		Items:        []br.Product{{Description: "Consultoria 💻", Amount: 10}},
	})

	if err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}
