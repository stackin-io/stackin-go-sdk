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

func ptr[T any](v T) *T {
	return &v
}

// gs1CheckDigit computes the GS1 modulo-10 check digit of body
// independently of the code under test, to build vectors in the test.
func gs1CheckDigit(body string) byte {
	sum := 0
	weight := 3
	for i := len(body) - 1; i >= 0; i-- {
		sum += int(body[i]-'0') * weight
		weight = 4 - weight
	}
	return byte('0' + (10-sum%10)%10)
}

func TestGS1CheckDigitHelperMatchesPublishedVectors(t *testing.T) {
	for _, gtin := range []string{"4006381333931", "96385074", "036000291452", "7891000100103"} {
		body := gtin[:len(gtin)-1]
		if got := gs1CheckDigit(body); got != gtin[len(gtin)-1] {
			t.Errorf("gs1CheckDigit(%q) = %c, want %c", body, got, gtin[len(gtin)-1])
		}
	}
}

func TestValidateProductAcceptsAValidGTINCheckDigit(t *testing.T) {
	gtins := []string{"4006381333931", "96385074", "036000291452", "7891000100103"}
	gtin14 := "1789123456789"
	gtins = append(gtins, gtin14+string(gs1CheckDigit(gtin14)))
	for _, gtin := range gtins {
		if err := validateProduct(br.Product{Description: "Item", Barcode: ptr(gtin)}); err != nil {
			t.Errorf("validateProduct(barcode=%q) = %v, want nil", gtin, err)
		}
	}
}

func TestValidateProductRejectsAWrongGTINCheckDigit(t *testing.T) {
	// 17891234567894 is the issue's example; its check digit is 2.
	for _, gtin := range []string{"17891234567894", "4006381333932", "96385075", "036000291453", "7891000100100"} {
		err := validateProduct(br.Product{Description: "Item", Barcode: ptr(gtin)})
		if err == nil || err.Error() != "barcode has an invalid GTIN check digit" {
			t.Errorf("validateProduct(barcode=%q) = %v, want the check digit error", gtin, err)
		}
	}
}

func TestValidateProductSkipsTheCheckDigitOnSemGTIN(t *testing.T) {
	if err := validateProduct(br.Product{Description: "Item", Barcode: ptr("SEM GTIN")}); err != nil {
		t.Errorf("validateProduct(SEM GTIN) = %v, want nil", err)
	}
}
