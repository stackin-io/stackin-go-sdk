package stackin

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const ibsCbsReply = `{
  "simulated": true,
  "notice": "simulation",
  "document_type": "nfe",
  "operation_date": "2026-03-10",
  "items": [{
    "cst": "000",
    "classification": "000001",
    "indicators": {"group": null, "reduction": 1, "deferral": null},
    "has_group": true,
    "rates": {"rate_state": "0.10", "rate_city": "0.00", "rate_federal": "0.90"},
    "rates_source": "official_2026",
    "reduction_ibs": null,
    "reduction_cbs": "60.00",
    "effective_rates": {"rate_state": "0.10", "rate_city": "0.00", "rate_federal": "0.36"},
    "deferral": null,
    "future_field": 1,
    "amounts": {"base": "100.00", "ibs_state": "0.10", "ibs_city": "0.00", "ibs": "0.10", "cbs": "0.36",
                "deferred_ibs_state": "0.00", "deferred_ibs_city": "0.00", "deferred_cbs": "0.00"}
  }],
  "total": {"base": "100.00", "ibs_state": "0.10", "ibs_city": "0.00", "ibs": "0.10", "cbs": "0.36",
            "deferred_ibs_state": "0.00", "deferred_ibs_city": "0.00", "deferred_cbs": "0.00"}
}`

func pct(v float64) *float64 { return &v }

func validIbsCbsItem() IbsCbsCalculationItem {
	return IbsCbsCalculationItem{CST: "000", Classification: "000001", Amount: pct(100)}
}

func TestCalculateIbsCbsSendsOnlyWhatIsSet(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, ibsCbsReply)

	_, err := NewFiscalReference(WithBaseURL(url), WithAPIKey("k")).CalculateIbsCbs(IbsCbsCalculationRequest{
		OperationDate: "2026-03-10",
		Items:         []IbsCbsCalculationItem{validIbsCbsItem()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.uri != "/api/v1/ibs-cbs/calculate" {
		t.Errorf("request = %s %s", got.method, got.uri)
	}
	want := map[string]any{
		"operation_date": "2026-03-10",
		"items": []any{
			map[string]any{"cst": "000", "classification": "000001", "amount": 100.0},
		},
	}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
}

func TestCalculateIbsCbsSendsRatesAndZeroValues(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, ibsCbsReply)

	item := validIbsCbsItem()
	item.RateCity = pct(0)
	item.ReductionCbs = pct(60)
	_, err := NewFiscalReference(WithBaseURL(url), WithAPIKey("k")).CalculateIbsCbs(IbsCbsCalculationRequest{
		DocumentType:  "nfse",
		OperationDate: "2027-01-05",
		Rates:         &IbsCbsRates{RateState: pct(0.1), RateFederal: pct(0.9)},
		Items:         []IbsCbsCalculationItem{item},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"document_type":  "nfse",
		"operation_date": "2027-01-05",
		"rates":          map[string]any{"rate_state": 0.1, "rate_federal": 0.9},
		"items": []any{
			map[string]any{
				"cst": "000", "classification": "000001", "amount": 100.0,
				"rate_city": 0.0, "reduction_cbs": 60.0,
			},
		},
	}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
}

func TestCalculateIbsCbsDecodesTheTypedResponse(t *testing.T) {
	url, _ := routeServer(t, http.StatusOK, ibsCbsReply)

	result, err := NewFiscalReference(WithBaseURL(url), WithAPIKey("k")).CalculateIbsCbs(IbsCbsCalculationRequest{
		OperationDate: "2026-03-10",
		Items:         []IbsCbsCalculationItem{validIbsCbsItem()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Simulated || result.DocumentType != "nfe" || result.OperationDate != "2026-03-10" {
		t.Errorf("result = %+v", result)
	}
	item := result.Items[0]
	if item.Indicators.Group != nil || item.Indicators.Reduction == nil || *item.Indicators.Reduction != 1 {
		t.Errorf("indicators = %+v", item.Indicators)
	}
	if !item.HasGroup || item.RatesSource == nil || *item.RatesSource != "official_2026" {
		t.Errorf("item = %+v", item)
	}
	if item.Rates == nil || item.Rates.RateFederal != "0.90" || item.EffectiveRates.RateFederal != "0.36" {
		t.Errorf("rates = %+v / %+v", item.Rates, item.EffectiveRates)
	}
	if item.ReductionIbs != nil || item.ReductionCbs == nil || *item.ReductionCbs != "60.00" {
		t.Errorf("reductions = %v / %v", item.ReductionIbs, item.ReductionCbs)
	}
	if item.Amounts.Cbs != "0.36" || result.Total.Base != "100.00" || result.Total.Ibs != "0.10" {
		t.Errorf("amounts = %+v / %+v", item.Amounts, result.Total)
	}
}

func TestCalculateIbsCbsRefusesLocally(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, ibsCbsReply)
	ref := NewFiscalReference(WithBaseURL(url), WithAPIKey("k"))

	many := make([]IbsCbsCalculationItem, 991)
	for i := range many {
		many[i] = validIbsCbsItem()
	}
	item := func(change func(*IbsCbsCalculationItem)) []IbsCbsCalculationItem {
		i := validIbsCbsItem()
		change(&i)
		return []IbsCbsCalculationItem{i}
	}
	ok := []IbsCbsCalculationItem{validIbsCbsItem()}

	cases := []struct {
		name     string
		request  IbsCbsCalculationRequest
		fragment string
	}{
		{"no items", IbsCbsCalculationRequest{OperationDate: "2026-03-10"}, "items must have"},
		{"too many items", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: many}, "items must have"},
		{"bad date", IbsCbsCalculationRequest{OperationDate: "10/03/2026", Items: ok}, "operation_date"},
		{"no date", IbsCbsCalculationRequest{Items: ok}, "operation_date"},
		{"document type", IbsCbsCalculationRequest{DocumentType: "cte", OperationDate: "2026-03-10", Items: ok}, "document_type"},
		{"cst", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.CST = "00" })}, "cst"},
		{"classification", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.Classification = "1" })}, "classification"},
		{"amount or base", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.Amount = nil })}, "amount or base"},
		{"amount zero", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.Amount = pct(0) })}, "amount must be greater"},
		{"base zero", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.Base = pct(-1) })}, "base must be greater"},
		{"item rate", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.RateState = pct(100.5) })}, "rate_state"},
		{"reduction", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Items: item(func(i *IbsCbsCalculationItem) { i.ReductionIbs = pct(-1) })}, "reduction_ibs"},
		{"rates", IbsCbsCalculationRequest{OperationDate: "2026-03-10", Rates: &IbsCbsRates{RateCity: pct(101)}, Items: ok}, "rates.rate_city"},
	}
	for _, c := range cases {
		_, err := ref.CalculateIbsCbs(c.request)
		var invErr *InvoiceError
		if !errors.As(err, &invErr) || !strings.Contains(invErr.Message, c.fragment) {
			t.Errorf("%s: err = %v, want a refusal mentioning %q", c.name, err, c.fragment)
		}
	}
	if got.calls != 0 {
		t.Errorf("calls = %d, want 0", got.calls)
	}
}

func TestCalculateIbsCbsAPIRefusalIsAnAPIError(t *testing.T) {
	url, _ := routeServer(t, http.StatusUnprocessableEntity, `{"detail":{"code":"IBS_CBS_RATES_REQUIRED","message":"rates required","field":"rates"}}`)

	_, err := NewFiscalReference(WithBaseURL(url), WithAPIKey("k")).CalculateIbsCbs(IbsCbsCalculationRequest{
		OperationDate: "2027-03-10",
		Items:         []IbsCbsCalculationItem{validIbsCbsItem()},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("err = %v, want *APIError 422", err)
	}
}
