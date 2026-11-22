package stackin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const maxIbsCbsItems = 990

var (
	cstPattern            = regexp.MustCompile(`^\d{3}$`)
	classificationPattern = regexp.MustCompile(`^\d{6}$`)
)

// IbsCbsRates are the caller's rates for the whole simulation, in percent
// (0 to 100): pIBSUF, pIBSMun and pCBS. Each is sent only when set, and an
// item's own rate wins over it. A nil IbsCbsCalculationRequest.Rates in
// 2026 makes the official 2026 rates apply (CBS 0.9%, IBS UF 0.1%, IBS
// Mun 0%); outside 2026 the API refuses it with IBS_CBS_RATES_REQUIRED.
type IbsCbsRates struct {
	RateState   *float64
	RateCity    *float64
	RateFederal *float64
}

// IbsCbsCalculationItem is one item of the simulation. CST (3 digits) and
// Classification (cClassTrib, 6 digits) are required, and so is Amount or
// Base. Fields left nil are not sent.
type IbsCbsCalculationItem struct {
	CST            string
	Classification string
	Amount         *float64
	Base           *float64
	RateState      *float64
	RateCity       *float64
	RateFederal    *float64
	ReductionIbs   *float64
	ReductionCbs   *float64
}

// IbsCbsCalculationRequest is what CalculateIbsCbs sends. OperationDate is
// YYYY-MM-DD and required. DocumentType is "nfe" (the default when
// empty), "nfce" or "nfse": it only picks the model the cClassTrib is
// checked for.
type IbsCbsCalculationRequest struct {
	DocumentType  string
	OperationDate string
	Rates         *IbsCbsRates
	Items         []IbsCbsCalculationItem
}

// IbsCbsIndicators are the cClassTrib's group, reduction and deferral
// indicators; nil when the classification does not carry one.
type IbsCbsIndicators struct {
	Group     *int `json:"group"`
	Reduction *int `json:"reduction"`
	Deferral  *int `json:"deferral"`
}

// IbsCbsAppliedRates are rates in percent, as decimal strings.
type IbsCbsAppliedRates struct {
	RateState   string `json:"rate_state"`
	RateCity    string `json:"rate_city"`
	RateFederal string `json:"rate_federal"`
}

// IbsCbsAmounts are money values as the API sends them, decimal strings
// such as "1.00", so nothing is rounded through a float.
type IbsCbsAmounts struct {
	Base             string `json:"base"`
	IbsState         string `json:"ibs_state"`
	IbsCity          string `json:"ibs_city"`
	Ibs              string `json:"ibs"`
	Cbs              string `json:"cbs"`
	DeferredIbsState string `json:"deferred_ibs_state"`
	DeferredIbsCity  string `json:"deferred_ibs_city"`
	DeferredCbs      string `json:"deferred_cbs"`
}

// IbsCbsCalculatedItem is one simulated item. RatesSource is "caller" or
// "official_2026"; Rates and EffectiveRates are nil when the item has
// none.
type IbsCbsCalculatedItem struct {
	CST            string              `json:"cst"`
	Classification string              `json:"classification"`
	Indicators     IbsCbsIndicators    `json:"indicators"`
	HasGroup       bool                `json:"has_group"`
	Rates          *IbsCbsAppliedRates `json:"rates"`
	RatesSource    *string             `json:"rates_source"`
	ReductionIbs   *string             `json:"reduction_ibs"`
	ReductionCbs   *string             `json:"reduction_cbs"`
	EffectiveRates *IbsCbsAppliedRates `json:"effective_rates"`
	Deferral       *string             `json:"deferral"`
	Amounts        IbsCbsAmounts       `json:"amounts"`
}

// IbsCbsCalculation is the simulation. Simulated is always true: nothing
// is issued or stored.
type IbsCbsCalculation struct {
	Simulated     bool                   `json:"simulated"`
	Notice        string                 `json:"notice"`
	DocumentType  string                 `json:"document_type"`
	OperationDate string                 `json:"operation_date"`
	Items         []IbsCbsCalculatedItem `json:"items"`
	Total         IbsCbsAmounts          `json:"total"`
}

// CalculateIbsCbs simulates the IBS and CBS of up to 990 items
// (POST /ibs-cbs/calculate). The amounts are the NF-e's: the simulation
// does not compute an NFS-e's, which the Sefin Nacional does. It needs no
// company and writes nothing. A refusal by the API, such as
// IBS_CBS_RATES_REQUIRED, is an *APIError with its code and field.
func (r *FiscalReference) CalculateIbsCbs(request IbsCbsCalculationRequest) (*IbsCbsCalculation, error) {
	body, err := request.body()
	if err != nil {
		return nil, err
	}

	raw, err := r.send(http.MethodPost, "/ibs-cbs/calculate", body, nil)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, &InvoiceError{Message: "unexpected response shape: empty body"}
	}
	var calculation IbsCbsCalculation
	if err := json.Unmarshal(raw, &calculation); err != nil {
		return nil, &InvoiceError{Message: "unexpected response shape: " + err.Error()}
	}
	return &calculation, nil
}

func (q IbsCbsCalculationRequest) body() (map[string]any, error) {
	if q.DocumentType != "" && q.DocumentType != "nfe" && q.DocumentType != "nfce" && q.DocumentType != "nfse" {
		return nil, &InvoiceError{Message: "document_type must be nfe, nfce or nfse"}
	}
	if _, err := time.Parse("2006-01-02", q.OperationDate); err != nil {
		return nil, &InvoiceError{Message: "operation_date must be a YYYY-MM-DD date"}
	}
	if len(q.Items) < 1 || len(q.Items) > maxIbsCbsItems {
		return nil, &InvoiceError{Message: fmt.Sprintf("items must have 1 to %d entries", maxIbsCbsItems)}
	}

	body := map[string]any{"operation_date": q.OperationDate}
	if q.DocumentType != "" {
		body["document_type"] = q.DocumentType
	}
	if q.Rates != nil {
		rates := map[string]any{}
		for _, field := range []struct {
			key   string
			value *float64
		}{
			{"rate_state", q.Rates.RateState},
			{"rate_city", q.Rates.RateCity},
			{"rate_federal", q.Rates.RateFederal},
		} {
			if err := checkPercent("rates."+field.key, field.value); err != nil {
				return nil, &InvoiceError{Message: err.Error()}
			}
			if field.value != nil {
				rates[field.key] = *field.value
			}
		}
		if len(rates) > 0 {
			body["rates"] = rates
		}
	}

	items := make([]map[string]any, 0, len(q.Items))
	for i, item := range q.Items {
		entry, err := item.body()
		if err != nil {
			return nil, &InvoiceError{Message: fmt.Sprintf("items[%d]: %s", i, err.Error())}
		}
		items = append(items, entry)
	}
	body["items"] = items
	return body, nil
}

func (i IbsCbsCalculationItem) body() (map[string]any, error) {
	if !cstPattern.MatchString(i.CST) {
		return nil, fmt.Errorf("cst must be 3 digits")
	}
	if !classificationPattern.MatchString(i.Classification) {
		return nil, fmt.Errorf("classification must be 6 digits")
	}
	if i.Amount == nil && i.Base == nil {
		return nil, fmt.Errorf("amount or base is required")
	}
	entry := map[string]any{"cst": i.CST, "classification": i.Classification}
	for _, field := range []struct {
		key   string
		value *float64
	}{{"amount", i.Amount}, {"base", i.Base}} {
		if field.value == nil {
			continue
		}
		if *field.value <= 0 {
			return nil, fmt.Errorf("%s must be greater than zero", field.key)
		}
		entry[field.key] = *field.value
	}
	for _, field := range []struct {
		key   string
		value *float64
	}{
		{"rate_state", i.RateState},
		{"rate_city", i.RateCity},
		{"rate_federal", i.RateFederal},
		{"reduction_ibs", i.ReductionIbs},
		{"reduction_cbs", i.ReductionCbs},
	} {
		if err := checkPercent(field.key, field.value); err != nil {
			return nil, err
		}
		if field.value != nil {
			entry[field.key] = *field.value
		}
	}
	return entry, nil
}

func checkPercent(key string, value *float64) error {
	if value != nil && (*value < 0 || *value > 100) {
		return fmt.Errorf("%s must be between 0 and 100", key)
	}
	return nil
}
