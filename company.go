package stackin

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var specialTaxRegimes = map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "9": true}

// Company fixes the issuer itself: its address and, for the owner, its
// fiscal fields (PATCH /company). It fixes what Invoice.Readiness reports
// missing; reading stays Readiness.
//
// It is company data: WithCompany and ForCompany pick the company. On an
// OAuth credential Update needs company:write; an API key acts for the
// member who created it.
type Company struct {
	transport
}

func NewCompany(opts ...Option) *Company {
	return &Company{transport: newTransport(opts...)}
}

// ForCompany returns a copy of the client bound to another company, for
// the calls chained on it. The receiver is left as it was.
func (c *Company) ForCompany(company string) *Company {
	bound := *c
	bound.transport.Company = company
	return &bound
}

// CompanyUpdate is what Update sends. A nil group, or one with nothing
// set, is left out; an empty CompanyUpdate changes nothing and answers
// the company as it is.
type CompanyUpdate struct {
	Address *CompanyAddress
	// FiscalFields are the company owner's alone: anyone else gets an
	// *APIError 403, even when the address in the same call would pass.
	FiscalFields          *FiscalFields
	BlockActivityMismatch *bool
}

// CompanyAddress is the issuer address Update may fix. There is no
// complement, and the state changes only in the dashboard: it decides the
// authorizer. Empty fields are left out.
type CompanyAddress struct {
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	// MunicipalityCode is the 7-digit IBGE code of a city in the company's
	// state. MunicipalityName is derived from it when left out.
	MunicipalityCode string `json:"municipality_code,omitempty"`
	// MunicipalityName is resolved to its code when sent alone, and must be
	// the code's city when both are sent.
	MunicipalityName string `json:"municipality_name,omitempty"`
	// ZipCode is the 8-digit CEP; a mask (01310-100) is stripped.
	ZipCode string `json:"zip_code,omitempty"`
}

// FiscalFields are the fiscal fields the owner may set. Empty fields are
// left out.
type FiscalFields struct {
	// TaxRegime is the CRT: 1 Simples, 2 Simples excesso, 3 normal, 4 MEI.
	TaxRegime string `json:"tax_regime,omitempty"`
	// SimplesNacional is the NFS-e opSimpNac: 1 no, 2 MEI, 3 ME/EPP.
	SimplesNacional string `json:"simples_nacional,omitempty"`
	// SimplesApuracao is the NFS-e regApTribSN for ME/EPP, 1 to 3.
	SimplesApuracao string `json:"simples_apuracao,omitempty"`
	// SimplesApuracaoIbsCbs is the apuração of the IBS/CBS for an ME/EPP,
	// 1 or 3; 3 is what the NFS-e IBS/CBS group needs.
	SimplesApuracaoIbsCbs string `json:"simples_apuracao_ibs_cbs,omitempty"`
	// StateRegistration is the inscrição estadual (IE), or ISENTO.
	StateRegistration string `json:"state_registration,omitempty"`
	// ServiceCode is the default cTribNac, e.g. 1.07 or 01.07.00.
	ServiceCode string `json:"service_code,omitempty"`

	// The fields below are pointers: nil leaves the setting alone and a
	// pointer to "" clears it.
	//
	// The ApproximateTax* settings are the Lei 12.741 percentages (DPS
	// totTrib), two decimals: the Simples one 0 to 99.99, the others 0 to
	// 100. IssRate is the ISSQN rate in percent, above 0 and at most 5.
	// MunicipalRegistration is the provider's inscrição municipal, up to 15
	// letters or digits. SpecialTaxRegime is regEspTrib: 0, 1, 2, 3, 4, 5, 6
	// or 9.
	ApproximateTaxSimplesPercent   *string `json:"approximate_tax_simples_percent,omitempty"`
	ApproximateTaxFederalPercent   *string `json:"approximate_tax_federal_percent,omitempty"`
	ApproximateTaxStatePercent     *string `json:"approximate_tax_state_percent,omitempty"`
	ApproximateTaxMunicipalPercent *string `json:"approximate_tax_municipal_percent,omitempty"`
	IssRate                        *string `json:"iss_rate,omitempty"`
	MunicipalRegistration          *string `json:"municipal_registration,omitempty"`
	SpecialTaxRegime               *string `json:"special_tax_regime,omitempty"`
}

// Update fixes the issuer's address and, for the owner, its fiscal
// fields, and answers what GET /company does — ready, missing,
// by_document_type, fix_url, ... — plus the stored address and
// fiscal_fields, so no second call is needed.
//
// A municipality the IBGE table cannot be read for right now is an
// *APIError 503 with Retry-After: nothing was written. A member limited to
// homologation, or a non-owner sending FiscalFields, is an *APIError 403.
func (c *Company) Update(update CompanyUpdate) (map[string]any, error) {
	payload := map[string]any{}
	if update.Address != nil {
		address, err := update.Address.body()
		if err != nil {
			return nil, err
		}
		if len(address) > 0 {
			payload["address"] = address
		}
	}
	if update.FiscalFields != nil {
		fiscal, err := update.FiscalFields.body()
		if err != nil {
			return nil, err
		}
		if len(fiscal) > 0 {
			payload["fiscal_fields"] = fiscal
		}
	}
	if update.BlockActivityMismatch != nil {
		payload["block_activity_mismatch"] = *update.BlockActivityMismatch
	}
	return c.request(http.MethodPatch, "/company", payload, nil)
}

func (a *CompanyAddress) body() (map[string]string, error) {
	out := map[string]string{}
	put(out, "street", a.Street)
	put(out, "number", a.Number)
	put(out, "neighborhood", a.Neighborhood)
	if a.MunicipalityCode != "" {
		if !allDigits(a.MunicipalityCode, 7) {
			return nil, &InvoiceError{Message: "municipality_code must be the 7-digit IBGE code"}
		}
		out["municipality_code"] = a.MunicipalityCode
	}
	put(out, "municipality_name", a.MunicipalityName)
	if a.ZipCode != "" {
		digits := strings.NewReplacer(" ", "", ".", "", "-", "").Replace(a.ZipCode)
		if !allDigits(digits, 8) {
			return nil, &InvoiceError{Message: "zip_code must be the 8-digit CEP"}
		}
		out["zip_code"] = digits
	}
	return out, nil
}

func (f *FiscalFields) body() (map[string]string, error) {
	out := map[string]string{}
	checks := []struct {
		key, value, allowed string
	}{
		{"simples_apuracao_ibs_cbs", f.SimplesApuracaoIbsCbs, "13"},
		{"tax_regime", f.TaxRegime, "1234"},
		{"simples_nacional", f.SimplesNacional, "123"},
		{"simples_apuracao", f.SimplesApuracao, "123"},
	}
	for _, check := range checks {
		if check.value == "" {
			continue
		}
		if len(check.value) != 1 || !strings.Contains(check.allowed, check.value) {
			return nil, &InvoiceError{Message: check.key + " must be one of " +
				strings.Join(strings.Split(check.allowed, ""), ", ")}
		}
		out[check.key] = check.value
	}
	put(out, "state_registration", f.StateRegistration)
	put(out, "service_code", f.ServiceCode)
	settings := []struct {
		key    string
		value  *string
		lo, hi float64
		loOpen bool
	}{
		{"approximate_tax_simples_percent", f.ApproximateTaxSimplesPercent, 0, 99.99, false},
		{"approximate_tax_federal_percent", f.ApproximateTaxFederalPercent, 0, 100, false},
		{"approximate_tax_state_percent", f.ApproximateTaxStatePercent, 0, 100, false},
		{"approximate_tax_municipal_percent", f.ApproximateTaxMunicipalPercent, 0, 100, false},
		{"iss_rate", f.IssRate, 0, 5, true},
	}
	for _, setting := range settings {
		if setting.value == nil {
			continue
		}
		text, err := percentSetting(setting.key, *setting.value, setting.lo, setting.hi, setting.loOpen)
		if err != nil {
			return nil, err
		}
		out[setting.key] = text
	}
	if f.MunicipalRegistration != nil {
		text := unmaskedRegistration(*f.MunicipalRegistration)
		if text != "" && !registrationPattern.MatchString(text) {
			return nil, &InvoiceError{Message: "municipal_registration takes up to 15 letters or digits"}
		}
		out["municipal_registration"] = text
	}
	if f.SpecialTaxRegime != nil {
		regime := strings.TrimSpace(*f.SpecialTaxRegime)
		if regime != "" && !specialTaxRegimes[regime] {
			return nil, &InvoiceError{Message: "special_tax_regime must be one of 0, 1, 2, 3, 4, 5, 6, 9"}
		}
		out["special_tax_regime"] = regime
	}
	return out, nil
}

func percentSetting(key, value string, lo, hi float64, loOpen bool) (string, error) {
	text := strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	if text == "" {
		return "", nil
	}
	number, err := strconv.ParseFloat(text, 64)
	span := fmt.Sprintf("from %s to %s", br.DecimalString(lo), br.DecimalString(hi))
	if loOpen {
		span = fmt.Sprintf("above %s and at most %s", br.DecimalString(lo), br.DecimalString(hi))
	}
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < lo || number > hi ||
		(loOpen && number == lo) || decimalPlaces(number) > 2 {
		return "", &InvoiceError{Message: fmt.Sprintf(
			"%s must be a percentage %s with at most two decimals; got %q", key, span, value)}
	}
	return br.DecimalString(number), nil
}

func put(out map[string]string, key, value string) {
	if value != "" {
		out[key] = value
	}
}

func allDigits(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
