package br

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	digits30        = regexp.MustCompile(`^[0-9]{30}$`)
	digits14        = regexp.MustCompile(`^[0-9]{14}$`)
	nfseKeyPattern  = regexp.MustCompile(`^[0-9]{6}[0-9A-Z]{14}[0-9]{30}$`)
	nfeKeyPattern   = regexp.MustCompile(`^[0-9]{6}[0-9A-Z]{14}[0-9]{24}$`)
	alnum15         = regexp.MustCompile(`^[A-Za-z0-9]{1,15}$`)
	municipalNfseNo = regexp.MustCompile(`^\d{1,15}$`)
	paperNumber     = regexp.MustCompile(`^\d{1,7}$`)
	verifyCode      = regexp.MustCompile(`^[A-Za-z0-9]{1,9}$`)
	currencyCode    = regexp.MustCompile(`^\d{3}$`)
	taxIDPattern    = regexp.MustCompile(`^(\d{11}|\d{14})$`)
	phonePattern    = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
	phoneMask       = strings.NewReplacer(" ", "", "(", "", ")", "", ".", "", "-", "")
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	intermediaryID  = regexp.MustCompile(`^([0-9A-Z]{12}\d{2}|\d{11})$`)

	percent     = bounds{lo: 0, hi: 100, places: 2}
	percentPos  = bounds{lo: 0, hi: 100, loOpen: true, places: 2}
	percentCap  = bounds{lo: 0, hi: 99.99, places: 2}
	money       = bounds{lo: 0, hi: amountMax, places: 2}
	moneyPos    = bounds{lo: 0, hi: amountMax, loOpen: true, places: 2}
	issRate     = bounds{lo: 0, hi: 5, loOpen: true, places: 2}
	percentOnly = bounds{lo: 0, hi: 100}
	moneyOnly   = bounds{lo: 0, hi: amountMax}

	pisCofinsCSTs = []string{
		"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
		"49", "50", "51", "52", "53", "54", "55", "56",
		"60", "61", "62", "63", "64", "65", "66", "67",
		"70", "71", "72", "73", "74", "75", "98", "99",
	}
	deductionTypes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 99, 101, 102, 103, 104, 105, 199}
	missingReasons = []int{1, 2}
)

// ApproximateTaxes is the approximate tax burden of Lei 12.741 (totTrib).
// Send exactly one shape: SimplesPercent; the federal, state and municipal
// percentages; or the federal, state and municipal amounts. A left-out
// member of the chosen group is written as 0 by the API.
type ApproximateTaxes struct {
	SimplesPercent   *float64
	FederalPercent   *float64
	StatePercent     *float64
	MunicipalPercent *float64
	FederalAmount    *float64
	StateAmount      *float64
	MunicipalAmount  *float64
}

func (t ApproximateTaxes) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.decimal("simples_percent", t.SimplesPercent, percentCap)
	w.decimal("federal_percent", t.FederalPercent, percent)
	w.decimal("state_percent", t.StatePercent, percent)
	w.decimal("municipal_percent", t.MunicipalPercent, percent)
	w.decimal("federal_amount", t.FederalAmount, money)
	w.decimal("state_amount", t.StateAmount, money)
	w.decimal("municipal_amount", t.MunicipalAmount, money)
	shapes := presentCount(
		t.SimplesPercent != nil,
		t.FederalPercent != nil || t.StatePercent != nil || t.MunicipalPercent != nil,
		t.FederalAmount != nil || t.StateAmount != nil || t.MunicipalAmount != nil,
	)
	switch {
	case shapes == 0:
		w.failWhole("needs simples_percent, the federal/state/municipal percentages or the federal/state/municipal amounts")
	case shapes > 1:
		w.failWhole("takes one shape: simples_percent, the percentages or the amounts")
	}
	return w.result()
}

// IssSuspension is exigSusp: Type is "judicial" or "administrative" and
// ProcessNumber the 30-digit proceeding number.
type IssSuspension struct {
	Type          string
	ProcessNumber string
}

func (s IssSuspension) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredChoice("type", s.Type, "judicial", "administrative")
	w.requiredPattern("process_number", s.ProcessNumber, digits30, "30 digits")
	return w.result()
}

// IssMunicipalBenefit is the municipality's registered benefit (BM): ID is
// the 14-digit nBM, and at most one of ReductionAmount and ReductionPercent
// goes with it.
type IssMunicipalBenefit struct {
	ID               string
	ReductionAmount  *float64
	ReductionPercent *float64
}

func (b IssMunicipalBenefit) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("id", b.ID, digits14, "14 digits")
	w.decimal("reduction_amount", b.ReductionAmount, money)
	w.decimal("reduction_percent", b.ReductionPercent, percentPos)
	if b.ReductionAmount != nil && b.ReductionPercent != nil {
		w.failWhole("takes reduction_amount or reduction_percent, not both")
	}
	return w.result()
}

// IssDetails is the ISSQN of the service (tribMun). Taxation is "taxable"
// (the default when empty), "immune", "export" or "not_levied"; Retention
// is "none", "tomador" or "intermediary".
type IssDetails struct {
	Taxation          string
	ImmunityType      *int
	ResultCountryCode string
	Suspension        *IssSuspension
	Rate              *float64
	Retention         string
	MunicipalBenefit  *IssMunicipalBenefit
}

func (d IssDetails) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.choice("taxation", d.Taxation, "taxable", "immune", "export", "not_levied")
	w.integer("immunity_type", d.ImmunityType, 1, 2, 3, 4, 5)
	w.pattern("result_country_code", d.ResultCountryCode, upperISO2, "an ISO 3166 alpha-2 code")
	w.decimal("rate", d.Rate, issRate)
	w.choice("retention", d.Retention, "none", "tomador", "intermediary")
	if d.Suspension != nil {
		body, err := d.Suspension.body(w.at("suspension"))
		w.adopt("suspension", body, err)
	}
	if d.MunicipalBenefit != nil {
		body, err := d.MunicipalBenefit.body(w.at("municipal_benefit"))
		w.adopt("municipal_benefit", body, err)
	}
	return w.result()
}

// PisCofins is the PIS and COFINS of the service. CST is required.
// WithholdingType is tpRetPisCofins, 0 to 9.
type PisCofins struct {
	CST             string
	Base            *float64
	PisRate         *float64
	CofinsRate      *float64
	PisAmount       *float64
	CofinsAmount    *float64
	WithholdingType *int
}

func (p PisCofins) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredChoice("cst", p.CST, pisCofinsCSTs...)
	w.decimal("base", p.Base, money)
	w.decimal("pis_rate", p.PisRate, percentCap)
	w.decimal("cofins_rate", p.CofinsRate, percentCap)
	w.decimal("pis_amount", p.PisAmount, money)
	w.decimal("cofins_amount", p.CofinsAmount, money)
	w.integer("withholding_type", p.WithholdingType, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	return w.result()
}

// FederalTaxes are the federal taxes of the service (tribFed). Send at
// least one member. The withheld amounts are in BRL and above 0.
type FederalTaxes struct {
	PisCofins                   *PisCofins
	SocialSecurityWithheld      *float64
	IrrfWithheld                *float64
	SocialContributionsWithheld *float64
}

func (f FederalTaxes) body(path string) (map[string]any, error) {
	w := newWire(path)
	if f.PisCofins != nil {
		body, err := f.PisCofins.body(w.at("pis_cofins"))
		w.adopt("pis_cofins", body, err)
	}
	w.decimal("social_security_withheld", f.SocialSecurityWithheld, moneyPos)
	w.decimal("irrf_withheld", f.IrrfWithheld, moneyPos)
	w.decimal("social_contributions_withheld", f.SocialContributionsWithheld, moneyPos)
	if f.PisCofins == nil && f.SocialSecurityWithheld == nil && f.IrrfWithheld == nil && f.SocialContributionsWithheld == nil {
		w.failWhole("needs pis_cofins, social_security_withheld, irrf_withheld or social_contributions_withheld")
	}
	return w.result()
}

// ServiceAddress is a Brazilian address of a construction, an event or a
// property: ZipCode (8 digits), Street, Number and Neighborhood are
// required.
type ServiceAddress struct {
	ZipCode      string
	Street       string
	Number       string
	Complement   string
	Neighborhood string
}

func (a ServiceAddress) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("zip_code", a.ZipCode, digits8, "8 digits")
	w.required("street", a.Street, 1, 255)
	w.required("number", a.Number, 1, 60)
	w.text("complement", a.Complement, 1, 156)
	w.required("neighborhood", a.Neighborhood, 1, 60)
	return w.result()
}

// Construction is the obra group. Identify the work by WorkID with
// WorkIDType ("cno" or "cib", a CIB being exactly 8 characters) or by
// Address, not both. PropertyRegistration is optional and not enough alone.
type Construction struct {
	PropertyRegistration string
	WorkIDType           string
	WorkID               string
	Address              *ServiceAddress
}

func (c Construction) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.text("property_registration", c.PropertyRegistration, 1, 30)
	w.choice("work_id_type", c.WorkIDType, "cno", "cib")
	w.text("work_id", c.WorkID, 1, 30)
	if c.Address != nil {
		body, err := c.Address.body(w.at("address"))
		w.adopt("address", body, err)
	}
	if has(c.WorkID) != has(c.WorkIDType) {
		w.failWhole("work_id and work_id_type go together: cno or cib, and the number")
	}
	switch {
	case !has(c.WorkID) && c.Address == nil:
		w.failWhole("needs work_id with work_id_type (CNO or CIB) or an address")
	case has(c.WorkID) && c.Address != nil:
		w.failWhole("takes work_id or address, not both")
	}
	if strings.TrimSpace(c.WorkIDType) == "cib" && len([]rune(strings.TrimSpace(c.WorkID))) != 8 {
		w.fail("work_id", "must have exactly 8 characters for a cib")
	}
	return w.result()
}

// Event is the atvEvento group. Name, StartDate and EndDate are required;
// identify the place by EventID or by Address, one of the two. The dates
// carry only their day.
type Event struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	EventID   string
	Address   *ServiceAddress
}

func (e Event) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.required("name", e.Name, 1, 255)
	w.requiredDay("start_date", e.StartDate)
	w.requiredDay("end_date", e.EndDate)
	w.text("event_id", e.EventID, 1, 30)
	if e.Address != nil {
		body, err := e.Address.body(w.at("address"))
		w.adopt("address", body, err)
	}
	if has(e.EventID) == (e.Address != nil) {
		w.failWhole("takes event_id or address, one of the two")
	}
	return w.result()
}

// References are the infoCompl references: OrderNumber (xPed), OrderItems
// (gItemPed, up to 99), ReferenceDocument (docRef) and TechnicalDocument
// (idDocTec). The free text stays in Product.Observations.
type References struct {
	OrderNumber       string
	OrderItems        []string
	ReferenceDocument string
	TechnicalDocument string
}

func (r References) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.text("order_number", r.OrderNumber, 1, 60)
	if len(r.OrderItems) > 0 {
		if len(r.OrderItems) > 99 {
			w.fail("order_items", "takes at most 99 entries")
		}
		items := make([]string, len(r.OrderItems))
		for i, item := range r.OrderItems {
			item = strings.TrimSpace(item)
			if n := len([]rune(item)); n < 1 || n > 60 {
				w.fail(fmt.Sprintf("order_items[%d]", i), "must be 1 to 60 characters")
			}
			items[i] = item
		}
		w.out["order_items"] = items
	}
	w.text("reference_document", r.ReferenceDocument, 1, 255)
	w.text("technical_document", r.TechnicalDocument, 1, 40)
	return w.result()
}

// PersonAddress is a person's address: in Brazil (CityCode and ZipCode) or
// abroad (CountryCode, PostalCode, City and State), never both. Street,
// Number and Neighborhood are required.
type PersonAddress struct {
	Street       string
	Number       string
	Complement   string
	Neighborhood string
	CityCode     string
	ZipCode      string
	CountryCode  string
	PostalCode   string
	City         string
	State        string
}

// DeductionSupplierAddress is the supplier's address (fornec/end).
type DeductionSupplierAddress = PersonAddress

// IntermediaryAddress is the intermediary's address (interm/end).
type IntermediaryAddress = PersonAddress

func (a PersonAddress) abroad() bool {
	return has(a.CountryCode)
}

func (a PersonAddress) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.required("street", a.Street, 1, 255)
	w.required("number", a.Number, 1, 60)
	w.text("complement", a.Complement, 1, 156)
	w.required("neighborhood", a.Neighborhood, 1, 60)
	w.pattern("city_code", a.CityCode, digits7, "7 digits")
	w.pattern("zip_code", a.ZipCode, digits8, "8 digits")
	w.pattern("country_code", a.CountryCode, upperISO2, "an ISO 3166 alpha-2 code")
	w.text("postal_code", a.PostalCode, 1, 11)
	w.text("city", a.City, 1, 60)
	w.text("state", a.State, 1, 60)
	national := has(a.CityCode) && has(a.ZipCode) && !has(a.CountryCode) && !has(a.PostalCode) && !has(a.City) && !has(a.State)
	foreign := has(a.CountryCode) && has(a.PostalCode) && has(a.City) && has(a.State) && !has(a.CityCode) && !has(a.ZipCode)
	switch {
	case foreign && strings.TrimSpace(a.CountryCode) == "BR":
		w.fail("country_code", "is for an address abroad and cannot be BR")
	case !national && !foreign:
		w.failWhole("takes city_code and zip_code for an address in Brazil, or country_code, postal_code, city and state for one abroad")
	}
	return w.result()
}

// DeductionSupplier is fornec, who issued the deducted document: TaxID (CPF
// or CNPJ, masks dropped) with an address in Brazil, or ForeignID (NIF)
// with an address abroad. Exactly one of the two.
type DeductionSupplier struct {
	TaxID     string
	ForeignID string
	Name      string
	Address   *PersonAddress
}

func (s DeductionSupplier) body(path string) (map[string]any, error) {
	w := newWire(path)
	if has(s.TaxID) {
		w.pattern("tax_id", unmasked(s.TaxID), taxIDPattern, "a CPF (11 digits) or a CNPJ (14 digits)")
	}
	w.text("foreign_id", s.ForeignID, 1, 40)
	w.required("name", s.Name, 1, 150)
	switch {
	case s.Address == nil:
		w.fail("address", "is required")
	default:
		body, err := s.Address.body(w.at("address"))
		w.adopt("address", body, err)
		if has(s.TaxID) && s.Address.abroad() {
			w.fail("address", "must be in Brazil, city_code and zip_code, for a supplier with tax_id")
		}
		if has(s.ForeignID) && !s.Address.abroad() {
			w.fail("address", "must be abroad, country_code, postal_code, city and state, for a supplier with foreign_id")
		}
	}
	if has(s.TaxID) == has(s.ForeignID) {
		w.failWhole("takes tax_id for a supplier in Brazil or foreign_id for one abroad, exactly one of them")
	}
	return w.result()
}

// MunicipalNfse is a municipal NFS-e outside the national repository.
type MunicipalNfse struct {
	CityCode         string
	Number           string
	VerificationCode string
}

func (m MunicipalNfse) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("city_code", m.CityCode, digits7, "7 digits")
	w.requiredPattern("number", m.Number, municipalNfseNo, "up to 15 digits")
	w.requiredPattern("verification_code", m.VerificationCode, verifyCode, "up to 9 letters or digits")
	return w.result()
}

// PaperInvoice is a paper service invoice (NFNFS).
type PaperInvoice struct {
	Number string
	Model  string
	Series string
}

func (p PaperInvoice) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("number", p.Number, paperNumber, "up to 7 digits")
	w.requiredPattern("model", p.Model, municipalNfseNo, "up to 15 digits")
	w.requiredPattern("series", p.Series, alnum15, "up to 15 letters or digits")
	return w.result()
}

// OtherFiscalDocument is a fiscal document outside the national
// repository. CityCode and Description are read by the homologation layout
// only.
type OtherFiscalDocument struct {
	Number      string
	CityCode    string
	Description string
}

func (d OtherFiscalDocument) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.required("number", d.Number, 1, 255)
	w.pattern("city_code", d.CityCode, digits7, "7 digits")
	w.text("description", d.Description, 1, 255)
	return w.result()
}

// OtherDocument is a non-fiscal document. Description is read by the
// homologation layout only.
type OtherDocument struct {
	Number      string
	Description string
}

func (d OtherDocument) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.required("number", d.Number, 1, 255)
	w.text("description", d.Description, 1, 255)
	return w.result()
}

// OtherDfe is a national-repository electronic document that is not an
// NFS-e, NF-e or CT-e (dFeNacional with tipoChaveDFe 9). Key is the
// chaveDFe, up to 50 characters, and Description says what the document is.
type OtherDfe struct {
	Key         string
	Description string
}

func (d OtherDfe) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.required("key", d.Key, 1, 50)
	w.required("description", d.Description, 1, 255)
	return w.result()
}

// DeductionDocument is one document behind the deduction (docDedRed, or
// docAjusteBC in the homologation layout). Set exactly one of NfseKey,
// NfeKey, CteKey, MunicipalNfse, PaperInvoice, FiscalDocument,
// OtherDocument and OtherDfe. Type is the tpDedRed code: 1 to 9, 99, 101 to 105 or 199.
type DeductionDocument struct {
	Type           int
	Description    string
	NfseKey        string
	NfeKey         string
	CteKey         string
	MunicipalNfse  *MunicipalNfse
	PaperInvoice   *PaperInvoice
	FiscalDocument *OtherFiscalDocument
	OtherDocument  *OtherDocument
	OtherDfe       *OtherDfe
	IssueDate      time.Time
	CompetenceDate time.Time
	TotalAmount    float64
	DeductedAmount float64
	Supplier       *DeductionSupplier
}

func (d DeductionDocument) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredInteger("type", d.Type, deductionTypes...)
	w.text("description", d.Description, 1, 150)
	w.pattern("nfse_key", d.NfseKey, nfseKeyPattern, "the 50-character access key of a national NFS-e")
	w.pattern("nfe_key", d.NfeKey, nfeKeyPattern, "the 44-character access key of an NF-e")
	w.pattern("cte_key", d.CteKey, nfeKeyPattern, "the 44-character access key of a CT-e")
	nested := []struct {
		key  string
		set  bool
		body func(string) (map[string]any, error)
	}{
		{"municipal_nfse", d.MunicipalNfse != nil, func(p string) (map[string]any, error) { return d.MunicipalNfse.body(p) }},
		{"paper_invoice", d.PaperInvoice != nil, func(p string) (map[string]any, error) { return d.PaperInvoice.body(p) }},
		{"fiscal_document", d.FiscalDocument != nil, func(p string) (map[string]any, error) { return d.FiscalDocument.body(p) }},
		{"other_document", d.OtherDocument != nil, func(p string) (map[string]any, error) { return d.OtherDocument.body(p) }},
		{"other_dfe", d.OtherDfe != nil, func(p string) (map[string]any, error) { return d.OtherDfe.body(p) }},
	}
	identified := presentCount(has(d.NfseKey), has(d.NfeKey), has(d.CteKey))
	for _, group := range nested {
		if group.set {
			identified++
			body, err := group.body(w.at(group.key))
			w.adopt(group.key, body, err)
		}
	}
	if identified != 1 {
		w.failWhole("takes exactly one of nfse_key, nfe_key, cte_key, municipal_nfse, paper_invoice, fiscal_document, other_document, other_dfe")
	}
	w.requiredDay("issue_date", d.IssueDate)
	w.day("competence_date", d.CompetenceDate)
	w.requiredDecimal("total_amount", d.TotalAmount, moneyPos)
	w.requiredDecimal("deducted_amount", d.DeductedAmount, moneyPos)
	if d.Supplier != nil {
		body, err := d.Supplier.body(w.at("supplier"))
		w.adopt("supplier", body, err)
	}
	return w.result()
}

// Deductions is the ISSQN base deduction or reduction (vDedRed, or
// vAjusteBC in the homologation layout). Percent and Amount are exclusive;
// Documents are 1 to 1000. Send at least one of the three.
type Deductions struct {
	Percent   *float64
	Amount    *float64
	Documents []DeductionDocument
}

func (d Deductions) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.decimal("percent", d.Percent, percentPos)
	w.decimal("amount", d.Amount, moneyPos)
	if len(d.Documents) > 1000 {
		w.fail("documents", "takes 1 to 1000 entries")
	}
	if len(d.Documents) > 0 {
		list := make([]map[string]any, 0, len(d.Documents))
		for i, document := range d.Documents {
			body, err := document.body(fmt.Sprintf("%s[%d]", w.at("documents"), i))
			if err != nil {
				w.err = firstError(w.err, err)
				break
			}
			list = append(list, body)
		}
		w.out["documents"] = list
	}
	switch {
	case d.Percent != nil && d.Amount != nil:
		w.failWhole("takes percent or amount, not both")
	case d.Percent == nil && d.Amount == nil && len(d.Documents) == 0:
		w.failWhole("needs percent, amount or documents")
	}
	return w.result()
}

func firstError(current, next error) error {
	if current != nil {
		return current
	}
	return next
}

// Substitution is the subst group: the access key of the authorized NFS-e
// this one replaces, the ReasonCode (01 to 05 or 99) and, with 99, a Reason
// of 15 to 255 Latin-1 characters.
type Substitution struct {
	AccessKey  string
	ReasonCode string
	Reason     string
}

func (s Substitution) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("access_key", s.AccessKey, nfseKeyPattern, "the 50-character access key of an NFS-e")
	code := strings.TrimSpace(s.ReasonCode)
	if len(code) == 1 && code[0] >= '0' && code[0] <= '9' {
		code = "0" + code
	}
	w.requiredChoice("reason_code", code, "01", "02", "03", "04", "05", "99")
	reason := strings.Join(strings.Fields(s.Reason), " ")
	if reason != "" {
		w.text("reason", reason, 15, 255)
		for _, r := range reason {
			if r > 0xFF {
				w.fail("reason", "takes Latin-1 characters only")
				break
			}
		}
	}
	if code == "99" && reason == "" {
		w.fail("reason", "is required with reason_code 99: 15 to 255 characters")
	}
	return w.result()
}

// NfseIntermediary is the interm group, identified by exactly one of
// TaxID (CPF or CNPJ, masks dropped), ForeignID (NIF) or
// ForeignIDMissingReason (1 exempt, 2 not required). A ForeignID or
// ForeignIDMissingReason needs an Address abroad, and a TaxID one in
// Brazil or none.
type NfseIntermediary struct {
	TaxID                  string
	ForeignID              string
	ForeignIDMissingReason *int
	Caepf                  string
	MunicipalRegistration  string
	Name                   string
	Address                *PersonAddress
	Phone                  string
	Email                  string
}

func (i NfseIntermediary) body(path string) (map[string]any, error) {
	w := newWire(path)
	if has(i.TaxID) {
		w.pattern("tax_id", strings.ToUpper(unmasked(i.TaxID)), intermediaryID, "a CPF (11 digits) or a CNPJ (14 characters)")
	}
	w.text("foreign_id", i.ForeignID, 1, 40)
	w.integer("foreign_id_missing_reason", i.ForeignIDMissingReason, missingReasons...)
	w.pattern("caepf", i.Caepf, digits14, "14 digits")
	if has(i.MunicipalRegistration) {
		w.pattern("municipal_registration", unmasked(i.MunicipalRegistration), alnum15, "up to 15 letters or digits")
	}
	w.required("name", i.Name, 1, 300)
	if has(i.Phone) {
		w.pattern("phone", phoneMask.Replace(strings.TrimSpace(i.Phone)), phonePattern, "8 to 15 digits with an optional leading +")
	}
	if has(i.Email) {
		w.pattern("email", i.Email, emailPattern, "an e-mail address")
	}
	if i.Address != nil {
		body, err := i.Address.body(w.at("address"))
		w.adopt("address", body, err)
	}
	if presentCount(has(i.TaxID), has(i.ForeignID), i.ForeignIDMissingReason != nil) != 1 {
		w.failWhole("takes exactly one of tax_id, foreign_id or foreign_id_missing_reason")
	}
	abroad := i.Address != nil && i.Address.abroad()
	if !has(i.TaxID) && !abroad {
		w.fail("address", "must be abroad (country_code, postal_code, city and state) for an intermediary with foreign_id or foreign_id_missing_reason")
	}
	if has(i.TaxID) && abroad {
		w.fail("address", "abroad needs foreign_id or foreign_id_missing_reason, not tax_id")
	}
	return w.result()
}

// NfseProvider is the DPS prest contact the issuing company adds: Caepf (14
// digits), Phone (8 to 15 digits with an optional leading +, masks dropped)
// and Email, all optional.
type NfseProvider struct {
	Caepf string
	Phone string
	Email string
}

func (p NfseProvider) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.pattern("caepf", p.Caepf, digits14, "14 digits")
	if has(p.Phone) {
		w.pattern("phone", phoneMask.Replace(strings.TrimSpace(p.Phone)), phonePattern, "8 to 15 digits with an optional leading +")
	}
	if has(p.Email) {
		w.pattern("email", p.Email, emailPattern, "an e-mail address")
	}
	return w.result()
}

// ForeignTrade is the comExt group, for a service traded with someone
// abroad. The coded fields take what the layout lists: ServiceMode 1 to 4,
// Relationship 0 to 6 or 9 (a pointer so 0, none, can be sent),
// ProviderSupport 1 to 8, RecipientSupport 1 to 26 and TemporaryGoods 1 to
// 3, where 2 needs ImportDeclaration and 3 needs ExportRegistration.
// CurrencyCode is the 3-digit BACEN code. ShareWithMdic is sent only when
// true.
type ForeignTrade struct {
	ServiceMode        int
	Relationship       *int
	CurrencyCode       string
	ForeignAmount      float64
	ProviderSupport    int
	RecipientSupport   int
	TemporaryGoods     int
	ImportDeclaration  string
	ExportRegistration string
	ShareWithMdic      bool
}

func (t ForeignTrade) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredInteger("service_mode", t.ServiceMode, 1, 2, 3, 4)
	if t.Relationship == nil {
		w.fail("relationship", "is required")
	}
	w.integer("relationship", t.Relationship, 0, 1, 2, 3, 4, 5, 6, 9)
	w.requiredPattern("currency_code", t.CurrencyCode, currencyCode, "3 digits")
	w.requiredDecimal("foreign_amount", t.ForeignAmount, moneyPos)
	w.requiredInteger("provider_support", t.ProviderSupport, 1, 2, 3, 4, 5, 6, 7, 8)
	recipient := make([]int, 26)
	for n := range recipient {
		recipient[n] = n + 1
	}
	w.requiredInteger("recipient_support", t.RecipientSupport, recipient...)
	w.requiredInteger("temporary_goods", t.TemporaryGoods, 1, 2, 3)
	w.text("import_declaration", t.ImportDeclaration, 1, 12)
	w.text("export_registration", t.ExportRegistration, 1, 12)
	switch {
	case t.TemporaryGoods == 1 && (has(t.ImportDeclaration) || has(t.ExportRegistration)):
		w.failWhole("temporary_goods 1 takes no import_declaration or export_registration")
	case t.TemporaryGoods == 2 && !has(t.ImportDeclaration):
		w.failWhole("temporary_goods 2 needs import_declaration")
	case t.TemporaryGoods == 3 && !has(t.ExportRegistration):
		w.failWhole("temporary_goods 3 needs export_registration")
	}
	if t.ShareWithMdic {
		w.out["share_with_mdic"] = true
	}
	return w.result()
}

// NfseDetails are the NFS-e document-level fields beyond the lines. Every
// group is optional and left out when nil; NFS-e only, refused for an NF-e.
// CompetenceDate (dCompet) carries only its day. ServiceCityCode and
// ServiceCountryCode are exclusive, and the country is never BR.
type NfseDetails struct {
	ApproximateTaxes   *ApproximateTaxes
	CompetenceDate     time.Time
	ServiceCityCode    string
	ServiceCountryCode string
	Iss                *IssDetails
	Federal            *FederalTaxes
	Deductions         *Deductions
	Substitution       *Substitution
	Construction       *Construction
	Event              *Event
	References         *References
	Provider           *NfseProvider
	RecipientCaepf     string
	Intermediary       *NfseIntermediary
	ForeignTrade       *ForeignTrade
}

// ToDict returns the group as the request body carries it, or the first
// shape rule it breaks. A group with nothing set is left out.
func (d NfseDetails) ToDict() (map[string]any, error) {
	w := newWire("nfse")
	w.day("competence_date", d.CompetenceDate)
	w.pattern("service_city_code", d.ServiceCityCode, digits7, "the 7-digit IBGE code")
	w.pattern("service_country_code", d.ServiceCountryCode, upperISO2, "an ISO 3166 alpha-2 code")
	if has(d.ServiceCityCode) && has(d.ServiceCountryCode) {
		w.failWhole("takes service_city_code or service_country_code, not both")
	}
	if strings.TrimSpace(d.ServiceCountryCode) == "BR" {
		w.fail("service_country_code", "is for a service performed abroad and cannot be BR")
	}
	w.pattern("recipient_caepf", d.RecipientCaepf, digits14, "14 digits")
	groups := []struct {
		key  string
		set  bool
		body func(string) (map[string]any, error)
	}{
		{"approximate_taxes", d.ApproximateTaxes != nil, func(p string) (map[string]any, error) { return d.ApproximateTaxes.body(p) }},
		{"iss", d.Iss != nil, func(p string) (map[string]any, error) { return d.Iss.body(p) }},
		{"federal", d.Federal != nil, func(p string) (map[string]any, error) { return d.Federal.body(p) }},
		{"deductions", d.Deductions != nil, func(p string) (map[string]any, error) { return d.Deductions.body(p) }},
		{"substitution", d.Substitution != nil, func(p string) (map[string]any, error) { return d.Substitution.body(p) }},
		{"construction", d.Construction != nil, func(p string) (map[string]any, error) { return d.Construction.body(p) }},
		{"event", d.Event != nil, func(p string) (map[string]any, error) { return d.Event.body(p) }},
		{"references", d.References != nil, func(p string) (map[string]any, error) { return d.References.body(p) }},
		{"provider", d.Provider != nil, func(p string) (map[string]any, error) { return d.Provider.body(p) }},
		{"intermediary", d.Intermediary != nil, func(p string) (map[string]any, error) { return d.Intermediary.body(p) }},
		{"foreign_trade", d.ForeignTrade != nil, func(p string) (map[string]any, error) { return d.ForeignTrade.body(p) }},
	}
	for _, group := range groups {
		if group.set {
			body, err := group.body(w.at(group.key))
			w.adopt(group.key, body, err)
		}
	}
	if has(d.ServiceCountryCode) {
		if d.Construction != nil && d.Construction.Address != nil {
			w.fail("construction.address", "is an address in Brazil, and a service performed abroad takes its address abroad: identify the construction by its code instead")
		}
		if d.Event != nil && d.Event.Address != nil {
			w.fail("event.address", "is an address in Brazil, and a service performed abroad takes its address abroad: identify the event by its code instead")
		}
	}
	return w.result()
}
