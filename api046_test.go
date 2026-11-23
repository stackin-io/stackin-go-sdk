package stackin

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

func asAPIError(t *testing.T, err error) *APIError {
	t.Helper()
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	return apiErr
}

func TestAPIErrorStructuredPreSefazDetail(t *testing.T) {
	url, _ := routeServer(t, 422, `{"detail":{"code":"NCM_NOT_FULL_CODE","field":"items[0].product.br.ncm","message":"ncm needs 8 digits","expected":"8 digits","received":"1234","sefaz_rejection":"777"}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Consult("k", NFE)
	e := asAPIError(t, err)
	if e.Code != "NCM_NOT_FULL_CODE" || e.Field != "items[0].product.br.ncm" || e.SefazRejection != "777" ||
		e.Expected != "8 digits" || e.Received != "1234" {
		t.Errorf("fields = %+v", e)
	}
	if e.Detail != "ncm needs 8 digits" {
		t.Errorf("Detail = %q", e.Detail)
	}
	if e.Retryable != nil {
		t.Error("Retryable set, want nil")
	}
}

func TestAPIErrorAuthorityBlock(t *testing.T) {
	url, _ := routeServer(t, 502, `{"detail":"authority did not answer","error":{"code":"OUTCOME_UNKNOWN","source":"authority","message":"m","retryable":false,"document_id":"DPS123"}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Consult("k", NFSE)
	e := asAPIError(t, err)
	if e.Code != "OUTCOME_UNKNOWN" || e.DocumentID != "DPS123" || e.Detail != "authority did not answer" {
		t.Errorf("fields = %+v", e)
	}
	if e.Retryable == nil || *e.Retryable {
		t.Errorf("Retryable = %v, want false", e.Retryable)
	}
}

func TestAPIErrorAuthorityRejectedFallsBackToTopLevelCode(t *testing.T) {
	url, _ := routeServer(t, 422, `{"detail":"rejected","code":"AUTHORITY_REJECTED"}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Consult("k", NFSE)
	if e := asAPIError(t, err); e.Code != "AUTHORITY_REJECTED" {
		t.Errorf("Code = %q", e.Code)
	}
}

func TestAPIErrorAuthorityBlockWinsOverTopLevelCode(t *testing.T) {
	url, _ := routeServer(t, 422, `{"detail":"rejected","code":"AUTHORITY_REJECTED","error":{"code":"539","retryable":true}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Consult("k", NFE)
	e := asAPIError(t, err)
	if e.Code != "539" || e.Retryable == nil || !*e.Retryable {
		t.Errorf("fields = %+v", e)
	}
}

func TestAPIErrorPydanticListLeavesCodeEmpty(t *testing.T) {
	url, _ := routeServer(t, 422, `{"detail":[{"loc":["body","x"],"msg":"bad"}]}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Consult("k", NFE)
	e := asAPIError(t, err)
	if e.Code != "" || e.Field != "" || e.Retryable != nil {
		t.Errorf("fields = %+v", e)
	}
}

func TestRateLimitedCodeReachesAPIError(t *testing.T) {
	url, _ := routeServer(t, 429, `{"code":"DISTRIBUTION_COOLDOWN","retry_at":"2030-01-01T00:00:00Z"}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).SyncReceived()
	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("err = %v", err)
	}
	if limited.Code != "DISTRIBUTION_COOLDOWN" || limited.APIError.Code != "DISTRIBUTION_COOLDOWN" {
		t.Errorf("codes = %q / %q", limited.Code, limited.APIError.Code)
	}
}

func TestPaymentRequiredCarriesCode(t *testing.T) {
	url, _ := routeServer(t, 402, `{"detail":{"code":"RECEIVED_PAYMENT_REQUIRED","message":"no credit"}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).ReceivedXML("K")
	var pay *PaymentRequiredError
	if !errors.As(err, &pay) {
		t.Fatalf("err = %v", err)
	}
	if pay.Code != "RECEIVED_PAYMENT_REQUIRED" || pay.Detail != "no credit" {
		t.Errorf("fields = %+v", pay.APIError)
	}
}

func TestReceivedXMLReturnsRawBytes(t *testing.T) {
	xml := "<?xml version=\"1.0\"?><nfeProc>\xe7\xe3o</nfeProc>"
	url, got := routeServer(t, http.StatusOK, xml)
	body, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).ReceivedXML("3519 0/..")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != xml {
		t.Errorf("body = %q", body)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/received-invoices/3519%200%2F../xml" {
		t.Errorf("%s %s", got.method, got.path)
	}
}

func TestReceivedXMLConflictCode(t *testing.T) {
	url, _ := routeServer(t, 409, `{"detail":{"code":"RECEIVED_XML_NOT_AVAILABLE","message":"summary only","manifestation":{"method":"POST","path":"/x"}}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).ReceivedXML("K")
	e := asAPIError(t, err)
	if e.StatusCode != 409 || e.Code != "RECEIVED_XML_NOT_AVAILABLE" {
		t.Errorf("fields = %+v", e)
	}
}

func TestReceivedXMLRefusesEmptyKey(t *testing.T) {
	_, err := NewInvoice(WithAPIKey("k")).ReceivedXML("..")
	wantRefusal(t, err, "path segment")
}

func TestConsultByDPSEscapesAndReturnsResult(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"result":{"status":"cancelled"}}`)
	res, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).ForCompany("c1").ConsultByDPS("DPS 1/2")
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "cancelled" {
		t.Errorf("res = %v", res)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/invoices/dps/DPS%201%2F2" || got.company != "c1" {
		t.Errorf("%s %s %s", got.method, got.path, got.company)
	}
	_, err = NewInvoice(WithAPIKey("k")).ConsultByDPS("")
	wantRefusal(t, err, "path segment")
}

func validateRequest() IssueRequest {
	return IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Acme",
		TaxID:        "12345678000195",
		Items:        []br.Product{{Description: "Svc", UnitPrice: ptr(10.0)}},
	}
}

func TestValidateValid(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"valid":true,"warnings":[{"code":"W1","message":"default"}]}`)
	req := validateRequest()
	req.IdempotencyKey = "abc"
	res, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Validate(req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid || len(res.Errors) != 0 || len(res.Warnings) != 1 || res.Warnings[0].Code != "W1" {
		t.Errorf("res = %+v", res)
	}
	if got.method != http.MethodPost || got.path != "/api/v1/invoices/validate" || got.body["document_type"] != "nfse" {
		t.Errorf("%s %s %v", got.method, got.path, got.body)
	}
	if _, leaked := got.body["idempotency_key"]; leaked {
		t.Error("idempotency key in body")
	}
}

func TestValidateRulesFailedIsAResult(t *testing.T) {
	url, _ := routeServer(t, 422, `{"valid":false,"detail":{"code":"A","field":"items[0].x","message":"m"},"errors":[{"code":"A","field":"items[0].x","message":"m","sefaz_rejection":"777","expected":"e","received":"r"},{"code":"B","field":null,"message":"blocker","fix_url":"/fix"}],"warnings":[]}`)
	res, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Validate(validateRequest())
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid || len(res.Errors) != 2 {
		t.Fatalf("res = %+v", res)
	}
	if res.Errors[0].SefazRejection != "777" || res.Errors[1].Field != "" || res.Errors[1].FixURL != "/fix" {
		t.Errorf("errors = %+v", res.Errors)
	}
}

func TestValidatePydanticRefusalRaises(t *testing.T) {
	url, _ := routeServer(t, 422, `{"detail":[{"loc":["body"],"msg":"bad"}]}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Validate(validateRequest())
	if e := asAPIError(t, err); e.StatusCode != 422 {
		t.Errorf("status = %d", e.StatusCode)
	}
}

func TestValidateOtherStatusesRaise(t *testing.T) {
	for _, status := range []int{401, 404, 429} {
		url, _ := routeServer(t, status, `{"detail":"no"}`)
		_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Validate(validateRequest())
		if e := asAPIError(t, err); e.StatusCode != status {
			t.Errorf("status = %d, want %d", e.StatusCode, status)
		}
	}
}

func TestValidateRefusesLocallyLikeIssue(t *testing.T) {
	_, err := NewInvoice(WithAPIKey("k")).Validate(IssueRequest{DocumentType: NFSE})
	wantRefusal(t, err, "items can't be empty")
}

func TestWebhookEventsIncludeReceivedInvoice(t *testing.T) {
	have := map[string]bool{}
	for _, e := range WebhookEvents {
		have[e] = true
	}
	for _, e := range []string{"received_invoice.stored", "received_invoice.full_document"} {
		if !have[e] {
			t.Errorf("missing %s", e)
		}
	}
}
