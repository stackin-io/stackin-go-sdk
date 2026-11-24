package stackin

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

func TestProductSendsNbsOnTheItemAndOmitsItWhenEmpty(t *testing.T) {
	with := br.Product{Description: "Svc", Amount: 10, NBS: "1.1502.10.00"}.ToDict()
	if with["nbs"] != "1.1502.10.00" {
		t.Errorf("nbs = %v", with["nbs"])
	}
	if _, inProduct := with["product"].(map[string]any)["nbs"]; inProduct {
		t.Error("nbs sent inside product")
	}
	if _, present := (br.Product{Description: "Svc", Amount: 10}).ToDict()["nbs"]; present {
		t.Error("nbs sent while unset")
	}
}

func TestIssueSendsNbsOnTheLine(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"result":{}}`)
	req := validateRequest()
	req.Items[0].NBS = "115021000"
	if _, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Issue(req); err != nil {
		t.Fatal(err)
	}
	line := got.body["items"].([]any)[0].(map[string]any)
	if line["nbs"] != "115021000" {
		t.Errorf("line = %v", line)
	}
}

func TestItemFieldsSendNbsAndOmitItWhenEmpty(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"item": map[string]any{}})
	fields := ItemFields{Kind: "service", Description: "Svc", NBS: "115021000"}
	if _, _, err := items.Upsert(fields); err != nil {
		t.Fatal(err)
	}
	if (*seen)[0].Body["nbs"] != "115021000" {
		t.Errorf("body = %v", (*seen)[0].Body)
	}
	fields.NBS = ""
	if _, _, err := items.Upsert(fields); err != nil {
		t.Fatal(err)
	}
	if _, present := (*seen)[1].Body["nbs"]; present {
		t.Error("nbs sent while unset")
	}
}

func TestItemChangesSendNbsAndNullToClear(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{})
	if _, err := items.Update(itemID, ItemChanges{NBS: text("1.1502.10.00")}); err != nil {
		t.Fatal(err)
	}
	if _, err := items.Update(itemID, ItemChanges{NBS: text("")}); err != nil {
		t.Fatal(err)
	}
	if _, err := items.Update(itemID, ItemChanges{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual((*seen)[0].Body, map[string]any{"nbs": "1.1502.10.00"}) {
		t.Errorf("set = %v", (*seen)[0].Body)
	}
	if v, present := (*seen)[1].Body["nbs"]; !present || v != nil {
		t.Errorf("clear = %v", (*seen)[1].Body)
	}
	if (*seen)[2].Raw != "{}" {
		t.Errorf("unset = %q", (*seen)[2].Raw)
	}
}

func TestIssueAddsTheWarningsToTheResult(t *testing.T) {
	url, _ := routeServer(t, http.StatusOK, `{"result":{"id":"i1"},"warnings":[{"code":"MEI_ACTIVITY_NOT_ALLOWED","message":"m","field":null},{"code":"ACTIVITY_NOT_REGISTERED","message":"n","field":"items[0]"}]}`)
	result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Issue(validateRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result["id"] != "i1" {
		t.Errorf("result = %v", result)
	}
	warnings, ok := result["warnings"].([]any)
	if !ok || len(warnings) != 2 {
		t.Fatalf("warnings = %v", result["warnings"])
	}
	first := warnings[0].(map[string]any)
	if first["code"] != "MEI_ACTIVITY_NOT_ALLOWED" || first["field"] != nil {
		t.Errorf("first = %v", first)
	}
	if warnings[1].(map[string]any)["field"] != "items[0]" {
		t.Errorf("second = %v", warnings[1])
	}
}

func TestIssueWarningsIsAnEmptyListWhenTheAPISendsNone(t *testing.T) {
	for _, reply := range []string{`{"result":{"id":"i1"},"warnings":[]}`, `{"result":{"id":"i1"}}`, `{"id":"i1"}`} {
		url, _ := routeServer(t, http.StatusOK, reply)
		result, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Issue(validateRequest())
		if err != nil {
			t.Fatal(err)
		}
		warnings, ok := result["warnings"].([]any)
		if !ok || len(warnings) != 0 {
			t.Errorf("%s: warnings = %#v", reply, result["warnings"])
		}
	}
}

func acceptLanguageServer(t *testing.T, reply map[string]any) (string, *[]seenRequest) {
	t.Helper()
	return captureServer(t, http.StatusOK, reply)
}

func TestIssueSendsAcceptLanguageAsAHeaderOnly(t *testing.T) {
	base, seen := acceptLanguageServer(t, map[string]any{"result": map[string]any{}})
	req := validateRequest()
	req.AcceptLanguage = "pt-BR"
	if _, err := NewInvoice(WithBaseURL(base), WithAPIKey("k")).Issue(req); err != nil {
		t.Fatal(err)
	}
	got := (*seen)[0]
	if got.Header.Get("Accept-Language") != "pt-BR" {
		t.Errorf("Accept-Language = %q", got.Header.Get("Accept-Language"))
	}
	for key := range got.Body {
		if key == "accept_language" || key == "AcceptLanguage" {
			t.Errorf("%s leaked into the body", key)
		}
	}
}

func TestValidateSendsAcceptLanguageAsAHeaderOnly(t *testing.T) {
	base, seen := acceptLanguageServer(t, map[string]any{"valid": true, "errors": []any{}, "warnings": []any{}})
	req := validateRequest()
	req.AcceptLanguage = "es"
	if _, err := NewInvoice(WithBaseURL(base), WithAPIKey("k")).Validate(req); err != nil {
		t.Fatal(err)
	}
	got := (*seen)[0]
	if got.Header.Get("Accept-Language") != "es" {
		t.Errorf("Accept-Language = %q", got.Header.Get("Accept-Language"))
	}
	for key := range got.Body {
		if key == "accept_language" || key == "AcceptLanguage" {
			t.Errorf("%s leaked into the body", key)
		}
	}
}

func TestAcceptLanguageIsNotSetByDefault(t *testing.T) {
	base, seen := acceptLanguageServer(t, map[string]any{"result": map[string]any{}})
	if _, err := NewInvoice(WithBaseURL(base), WithAPIKey("k")).Issue(validateRequest()); err != nil {
		t.Fatal(err)
	}
	if _, present := (*seen)[0].Header["Accept-Language"]; present {
		t.Error("Accept-Language sent while unset")
	}
}

func TestValidateExposesTheWarningField(t *testing.T) {
	url, _ := routeServer(t, http.StatusOK, `{"valid":true,"errors":[],"warnings":[{"code":"MEI_ACTIVITY_NOT_ALLOWED","message":"m","field":"items[0]"},{"code":"MEI_NO_ICMS_ACTIVITY","message":"n","field":null}]}`)
	res, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Validate(validateRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 2 || res.Warnings[0].Code != "MEI_ACTIVITY_NOT_ALLOWED" ||
		res.Warnings[0].Field != "items[0]" || res.Warnings[1].Field != "" {
		t.Errorf("warnings = %+v", res.Warnings)
	}
}

func TestCompanyUpdateSendsBlockActivityMismatchTrueAndFalse(t *testing.T) {
	for _, value := range []bool{true, false} {
		url, got := routeServer(t, http.StatusOK, `{"ready":true}`)
		v := value
		if _, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{BlockActivityMismatch: &v}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.body, map[string]any{"block_activity_mismatch": value}) {
			t.Errorf("body = %v, want block_activity_mismatch %v", got.body, value)
		}
	}
}

func TestCompanyUpdateLeavesBlockActivityMismatchOutWhenUnset(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"ready":true}`)
	if _, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{}); err != nil {
		t.Fatal(err)
	}
	if len(got.body) != 0 {
		t.Errorf("body = %v, want empty", got.body)
	}
}
