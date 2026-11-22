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

func ptr[T any](v T) *T {
	return &v
}

func TestValidateProductAcceptsWellFormedBarcodes(t *testing.T) {
	for _, barcode := range []string{"SEM GTIN", "96385074", "036000291452", "4006381333931", "17891000100100"} {
		if err := validateProduct(br.Product{Description: "Item", Barcode: ptr(barcode)}); err != nil {
			t.Errorf("validateProduct(barcode=%q) = %v, want nil", barcode, err)
		}
	}
}

func TestValidateProductRejectsMalformedBarcodes(t *testing.T) {
	for _, barcode := range []string{"", "SEM", "sem gtin", "1234567", "123456789", "12345678901", "123456789012345", "40063813339A1"} {
		err := validateProduct(br.Product{Description: "Item", Barcode: ptr(barcode)})
		want := "barcode must be 'SEM GTIN' or a GTIN of 8, 12, 13 or 14 digits"
		if err == nil || err.Error() != want {
			t.Errorf("validateProduct(barcode=%q) = %v, want %q", barcode, err, want)
		}
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

func TestValidateProductAcceptsAnEightDigitNCMOrTheServiceCode(t *testing.T) {
	for _, ncm := range []string{"84713012", "00"} {
		if err := validateProduct(br.Product{Description: "Notebook", NCM: ptr(ncm)}); err != nil {
			t.Errorf("validateProduct(ncm=%q) = %v, want nil", ncm, err)
		}
	}
}

func TestValidateProductAcceptsAProductWithoutNCM(t *testing.T) {
	if err := validateProduct(br.Product{Description: "Consultoria"}); err != nil {
		t.Errorf("validateProduct() = %v, want nil", err)
	}
}

func TestValidateProductRejectsAnyOtherNCM(t *testing.T) {
	for _, ncm := range []string{"84", "12", "000", "8471", "8471301", "847130120", "8471301A", " 84713012"} {
		err := validateProduct(br.Product{Description: "Notebook", NCM: ptr(ncm)})
		if err == nil || err.Error() != "ncm must be the full 8 digits, or 00 for a service line" {
			t.Errorf("validateProduct(ncm=%q) = %v, want the 8-digit error", ncm, err)
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

func TestIssueRejectsAWrongGTINCheckDigit(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	barcode := "17891234567894"
	item.Barcode = &barcode

	_, err := inv.Issue(nfeRequest(item))

	wantInvoiceError(t, err, *hits, "items[0].barcode has an invalid GTIN check digit")
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

func TestIssueRejectsAWrongRecopiCheckDigit(t *testing.T) {
	inv, hits := countingInvoice(t)
	item := nfeItem()
	item.RecopiNumber = ptr("20131106114609734394")

	_, err := inv.Issue(nfeRequest(item))

	wantInvoiceError(t, err, *hits, "items[0].recopi_number has invalid check digits")
}

func TestValidateProductAcceptsValidRecopiNumbers(t *testing.T) {
	// 20131106114609734393 is the MOC §8.5.3 worked example (DV1 9, DV2 3);
	// 00000000000012345600 has remainders 1 and 0, both giving DV 0.
	for _, recopi := range []string{"20131106114609734393", "00000000000012345600"} {
		if err := validateProduct(br.Product{Description: "Item", RecopiNumber: ptr(recopi)}); err != nil {
			t.Errorf("validateProduct(recopi_number=%q) = %v, want nil", recopi, err)
		}
	}
}

func TestValidateProductRejectsWrongRecopiCheckDigits(t *testing.T) {
	for _, recopi := range []string{"20131106114609734394", "20131106114609734383", "00000000000012345678"} {
		err := validateProduct(br.Product{Description: "Item", RecopiNumber: ptr(recopi)})
		if err == nil || err.Error() != "recopi_number has invalid check digits" {
			t.Errorf("validateProduct(recopi_number=%q) = %v, want the check digits error", recopi, err)
		}
	}
}

func TestValidateProductRejectsARecopiNumberThatIsNot20Digits(t *testing.T) {
	for _, recopi := range []string{"2013110611460973439", "201311061146097343930", "2013110611460973439A", ""} {
		err := validateProduct(br.Product{Description: "Item", RecopiNumber: ptr(recopi)})
		if err == nil || err.Error() != "recopi_number must have exactly 20 digits" {
			t.Errorf("validateProduct(recopi_number=%q) = %v, want the length error", recopi, err)
		}
	}
}

// nfeAccessKey is a well-formed 44-digit NF-e access key.
const nfeAccessKey = "35240912345678000199550010000000011000000017"

func TestCorrectRejectsAKeyThatIsNotAnNFEKey(t *testing.T) {
	keys := []string{
		"",
		"abc123",
		"3524091234567800019955001000000001100000001",        // 43 digits
		"352409123456780001995500100000000110000000170",      // 45 digits
		"35240912345678000199550010000000011000000017123456", // 50: an NFS-e key
		"3524091234567800019955001000000001100000001A",       // letter after the CNPJ slot
		"35240A12345678000199550010000000011000000017",       // letter before the CNPJ slot
		"352409abc45678000199550010000000011000000017",       // lowercase in the CNPJ slot
		"352409123456789ABCDE550010000000011000000017",       // letters at positions 19-20
		"35240912345678000199550010000000011000000017\n",     // trailing newline
	}
	for _, key := range keys {
		inv, hits := countingInvoice(t)

		_, err := inv.Correct(key, NFE, "Transportadora corrigida para Rapido Ltda")

		wantInvoiceError(t, err, *hits,
			"Correct is NFe only. NFSe has no correction letter — cancel and reissue instead.")
	}
}

func TestCorrectSendsAnNFEAccessKey(t *testing.T) {
	// 6 digits + 12 [0-9A-Z] (alphanumeric CNPJ) + 26 digits.
	alphanumeric := "352607" + "12ABC3450001" + strings.Repeat("9", 26)
	if len(alphanumeric) != 44 {
		t.Fatalf("test key has %d characters, want 44", len(alphanumeric))
	}
	for _, key := range []string{nfeAccessKey, alphanumeric} {
		inv, hits := countingInvoice(t)

		if _, err := inv.Correct(key, NFE, "Transportadora corrigida para Rapido Ltda"); err != nil {
			t.Fatalf("Correct(%q) = %v, want nil", key, err)
		}
		if *hits != 1 {
			t.Errorf("Correct(%q) hits = %d, want 1", key, *hits)
		}
	}
}

func TestCorrectLeavesAnNFSEDocumentTypeToTheAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "correction letters are NF-e only"})
	}))
	defer server.Close()
	inv := NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key"))

	_, err := inv.Correct(nfeAccessKey, NFSE, "Transportadora corrigida para Rapido Ltda")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("error = %v (%T), want *APIError with status 409", err, err)
	}
}

func TestIssueRequiresRecipientAddressForRetainedISSOnNFSE(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "11222333000181",
		Items: []br.Product{
			{Description: "Consultoria", Amount: 10},
			{Description: "Design", Amount: 10, TaxRetained: ptr(true)},
		},
	})

	wantInvoiceError(t, err, *hits,
		"recipient_address is required on an NFS-e when any item has tax_retained=true (ISSQN retained by the tomador)")
}

func TestIssueSendsRetainedISSOnNFSEWithAnAddress(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Design", Amount: 10, TaxRetained: ptr(true)}},
		RecipientAddress: validNFEAddress(),
	})

	if err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueRequiresACompleteAddressForRetainedISSOnNFSE(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Design", Amount: 10, TaxRetained: ptr(true)}},
		RecipientAddress: &Address{Street: "Rua das Flores", City: "Joinville", State: "SC"},
	})

	wantInvoiceError(t, err, *hits,
		"RecipientAddress is missing required fields for NFSE with tax_retained=true: Number, Neighborhood, ZipCode")
}

func TestIssueDoesNotAskForCityCodeOnRetainedISS(t *testing.T) {
	inv, hits := countingInvoice(t)
	address := validNFEAddress()
	address.CityCode = ""

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Design", Amount: 10, TaxRetained: ptr(true)}},
		RecipientAddress: address,
	})

	if err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestIssueLeavesAPartialAddressAloneWithoutRetainedISS(t *testing.T) {
	inv, hits := countingInvoice(t)

	_, err := inv.Issue(IssueRequest{
		DocumentType:     NFSE,
		ClientName:       "Acme",
		TaxID:            "11222333000181",
		Items:            []br.Product{{Description: "Design", Amount: 10}},
		RecipientAddress: &Address{City: "Joinville"},
	})

	if err != nil {
		t.Fatalf("Issue() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}
