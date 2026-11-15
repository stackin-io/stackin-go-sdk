package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestIssueRejectsAShortNCM(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	ncm := "84"
	item.NCM = &ncm

	_, err := inv.Issue(nfeRequest(item))

	wantInvoiceError(t, err, *hits, "items[0].ncm must be the full 8 digits, or 00 for a service line")
}

func TestIssueSendsAnEightDigitNCMOrTheServiceCode(t *testing.T) {
	for _, ncm := range []string{"84713012", "00"} {
		inv, hits := countingInvoice(t)
		item := nfeItem()
		item.NCM = &ncm

		if _, err := inv.Issue(nfeRequest(item)); err != nil {
			t.Fatalf("Issue(ncm=%q) = %v, want nil", ncm, err)
		}
		if *hits != 1 {
			t.Errorf("hits = %d, want 1", *hits)
		}
	}
}

func TestIssueRejectsAMalformedBarcode(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	barcode := "12345"
	item.Barcode = &barcode

	_, err := inv.Issue(nfeRequest(item))

	wantInvoiceError(t, err, *hits,
		"items[0].barcode must be 'SEM GTIN' or a GTIN of 8, 12, 13 or 14 digits")
}

func TestIssueRejectsAnNFEDescriptionOver120Characters(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	item.Description = strings.Repeat("a", 121)

	_, err := inv.Issue(nfeRequest(nfeItem(), item))

	wantInvoiceError(t, err, *hits, "items[1].description exceeds 120 characters (NF-e xProd limit)")
}

func TestIssueAcceptsAnNFEDescriptionOf120Characters(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	item.Description = strings.Repeat("ç", 120)

	if _, err := inv.Issue(nfeRequest(item)); err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueDoesNotLimitAnNFSEDescription(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "11222333000181",
		Items:        []br.Product{{Description: strings.Repeat("a", 500), Amount: 10}},
	})

	if err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueRejectsAShortStreetOrNeighborhood(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Address)
		message string
	}{
		{"street", func(a *Address) { a.Street = "R" }, "recipient_address.street must have at least 2 characters"},
		{"padded street", func(a *Address) { a.Street = "  R  " }, "recipient_address.street must have at least 2 characters"},
		{"neighborhood", func(a *Address) { a.Neighborhood = " " }, "recipient_address.neighborhood must have at least 2 characters"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv, hits := countingInvoice(t)
			req := nfeRequest(nfeItem())
			tc.mutate(req.RecipientAddress)

			_, err := inv.Issue(req)

			wantInvoiceError(t, err, *hits, tc.message)
		})
	}
}

func TestIssueChecksAddressLengthOnNFSEToo(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Consultoria", Amount: 10}},
		RecipientAddress: &Address{Street: "R", City: "Joinville"},
	})

	wantInvoiceError(t, err, *hits, "recipient_address.street must have at least 2 characters")
}

func TestAddressValidateAcceptsTwoCharactersAndAbsentFields(t *testing.T) {
	for _, address := range []Address{{}, {Street: "BR", Neighborhood: "Sé"}, *validNFEAddress()} {
		if err := address.validate(); err != nil {
			t.Errorf("validate(%+v) = %v, want nil", address, err)
		}
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
			"curly quote in client name",
			func(r *IssueRequest) { r.ClientName = "Acme “Ltda”" },
			"client_name contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '“'",
		},
		{
			"en dash in street",
			func(r *IssueRequest) { r.RecipientAddress.Street = "Rua A – Bloco B" },
			"recipient_address.street contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '–'",
		},
		{
			"tab in unit",
			func(r *IssueRequest) { r.Items[0].Unit = "U\tN" },
			"items[0].product.unit contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '\t'",
		},
		{
			"euro sign in purchase order",
			func(r *IssueRequest) { po := "PO-€1"; r.Items[0].PurchaseOrder = &po },
			"items[0].product.purchase_order contains a character outside ISO-8859-1 (latin1), which NF-e does not accept: '€'",
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
