package stackin

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestCompanyUpdateSendsBothGroups(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"ready":true,"missing":[],"fix_url":null,"address":{"state":"SP"},"fiscal_fields":{"tax_regime":"1"}}`)

	result, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{
		Address: &CompanyAddress{
			Street:           "Av. Paulista",
			Number:           "1000",
			Neighborhood:     "Bela Vista",
			MunicipalityCode: "3550308",
			MunicipalityName: "São Paulo",
			ZipCode:          "01310-100",
		},
		FiscalFields: &FiscalFields{
			TaxRegime:         "1",
			SimplesNacional:   "3",
			SimplesApuracao:   "1",
			StateRegistration: "ISENTO",
			ServiceCode:       "01.07.00",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPatch || got.uri != "/api/v1/company" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	want := map[string]any{
		"address": map[string]any{
			"street":            "Av. Paulista",
			"number":            "1000",
			"neighborhood":      "Bela Vista",
			"municipality_code": "3550308",
			"municipality_name": "São Paulo",
			"zip_code":          "01310100",
		},
		"fiscal_fields": map[string]any{
			"tax_regime":         "1",
			"simples_nacional":   "3",
			"simples_apuracao":   "1",
			"state_registration": "ISENTO",
			"service_code":       "01.07.00",
		},
	}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
	if result["ready"] != true || result["fix_url"] != nil {
		t.Errorf("result = %v", result)
	}
	if result["fiscal_fields"].(map[string]any)["tax_regime"] != "1" {
		t.Errorf("result = %v", result)
	}
}

func TestCompanyUpdateSendsOnlyWhatIsSet(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)

	_, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{
		Address:      &CompanyAddress{ZipCode: "01310 100"},
		FiscalFields: &FiscalFields{},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"address": map[string]any{"zip_code": "01310100"}}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
}

func TestCompanyUpdateWithNothingSendsAnEmptyObject(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{"ready":false}`)

	_, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{})
	if err != nil {
		t.Fatal(err)
	}
	if got.calls != 1 || got.body == nil || len(got.body) != 0 {
		t.Errorf("calls = %d, body = %v, want one call with {}", got.calls, got.body)
	}
}

func TestCompanyUpdateRefusesWhatTheAPIWould(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)
	company := NewCompany(WithBaseURL(url), WithAPIKey("k"))

	cases := []struct {
		update   CompanyUpdate
		fragment string
	}{
		{CompanyUpdate{Address: &CompanyAddress{MunicipalityCode: "355030"}}, "municipality_code"},
		{CompanyUpdate{Address: &CompanyAddress{MunicipalityCode: "35503O8"}}, "municipality_code"},
		{CompanyUpdate{Address: &CompanyAddress{ZipCode: "0131010"}}, "zip_code"},
		{CompanyUpdate{Address: &CompanyAddress{ZipCode: "01310/100"}}, "zip_code"},
		{CompanyUpdate{FiscalFields: &FiscalFields{TaxRegime: "5"}}, "tax_regime"},
		{CompanyUpdate{FiscalFields: &FiscalFields{SimplesNacional: "4"}}, "simples_nacional"},
		{CompanyUpdate{FiscalFields: &FiscalFields{SimplesApuracao: "12"}}, "simples_apuracao"},
	}
	for _, c := range cases {
		_, err := company.Update(c.update)
		wantRefusal(t, err, c.fragment)
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestCompanyForCompanySendsTheHeader(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)
	base := NewCompany(WithBaseURL(url), WithAPIKey("k"))

	if _, err := base.ForCompany("c-2").Update(CompanyUpdate{}); err != nil {
		t.Fatal(err)
	}
	if got.company != "c-2" {
		t.Errorf("company header = %q, want c-2", got.company)
	}
	if base.transport.Company != "" {
		t.Errorf("receiver changed: %q", base.transport.Company)
	}
}

func TestCompanyUpdateRefusalsAreAPIErrors(t *testing.T) {
	cases := []struct {
		status int
		reply  string
	}{
		{http.StatusForbidden, `{"detail":"Only the company owner can change fiscal fields."}`},
		{http.StatusServiceUnavailable, `{"detail":"The municipality could not be checked right now; nothing was changed. Retry in a moment."}`},
		{http.StatusUnprocessableEntity, `{"detail":[{"loc":["body","address","municipality_code"],"msg":"not an IBGE municipality"}]}`},
	}
	for _, c := range cases {
		url, _ := routeServer(t, c.status, c.reply)
		_, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{
			FiscalFields: &FiscalFields{TaxRegime: "1"},
		})
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != c.status {
			t.Errorf("status %d: err = %v, want *APIError", c.status, err)
		}
		var typed *InsufficientPermissionError
		if errors.As(err, &typed) {
			t.Errorf("status %d: typed as insufficient permission", c.status)
		}
	}
}

func TestCompanyUpdateWithoutTheScopeIsInsufficientPermission(t *testing.T) {
	url, _ := routeServer(t, http.StatusForbidden, `{"detail":{"error":"insufficient_scope","error_description":"INSUFFICIENT_PERMISSION: needs company:write","code":"INSUFFICIENT_PERMISSION"}}`)

	_, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{
		Address: &CompanyAddress{Street: "Rua A"},
	})
	var typed *InsufficientPermissionError
	if !errors.As(err, &typed) {
		t.Fatalf("err = %T, want *InsufficientPermissionError", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("err = %v, want an *APIError 403 inside", err)
	}
}
