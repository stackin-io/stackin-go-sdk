package br

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ibsCbsRecipientID = regexp.MustCompile(`^(\d{11}|[0-9A-Z]{12}\d{2})$`)
	digits6to20       = regexp.MustCompile(`^\d{6,20}$`)
	digits3           = regexp.MustCompile(`^\d{3}$`)
	digits6           = regexp.MustCompile(`^\d{6}$`)
	digits2           = regexp.MustCompile(`^\d{2}$`)
	cibPattern        = regexp.MustCompile(`^[0-9A-Z]{8}$`)
)

// IbsCbsGovernment is tpOper and tpEnteGov: OperationType "1" to "5" and
// EntityType "1" to "4". Send at least one.
type IbsCbsGovernment struct {
	OperationType string
	EntityType    string
}

func (g IbsCbsGovernment) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.choice("operation_type", g.OperationType, "1", "2", "3", "4", "5")
	w.choice("entity_type", g.EntityType, "1", "2", "3", "4")
	if !has(g.OperationType) && !has(g.EntityType) {
		w.failWhole("needs operation_type, entity_type or both")
	}
	return w.result()
}

// IbsCbsRecipientAddress is the dest/end address: in Brazil (CityCode and
// ZipCode) or abroad (CountryCode, PostalCode, City and Region), never
// both. Street, Number and Neighborhood are required.
type IbsCbsRecipientAddress struct {
	Street       string
	Number       string
	Complement   string
	Neighborhood string
	CityCode     string
	ZipCode      string
	CountryCode  string
	PostalCode   string
	City         string
	Region       string
}

func (a IbsCbsRecipientAddress) body(path string) (map[string]any, error) {
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
	w.text("region", a.Region, 1, 60)
	national := has(a.CityCode) || has(a.ZipCode)
	foreign := has(a.CountryCode) || has(a.PostalCode) || has(a.City) || has(a.Region)
	switch {
	case national && foreign:
		w.failWhole("is in Brazil (city_code, zip_code) or abroad (country_code, postal_code, city, region), not both")
	case foreign && !(has(a.CountryCode) && has(a.PostalCode) && has(a.City) && has(a.Region)):
		w.failWhole("abroad needs country_code, postal_code, city and region")
	case !foreign && !(has(a.CityCode) && has(a.ZipCode)):
		w.failWhole("in Brazil needs city_code and zip_code")
	}
	return w.result()
}

// IbsCbsRecipient is the dest group: who receives the service when it is
// not the tomador. Send exactly one of TaxID, NIF or NoNifReason ("0", "1"
// or "2").
type IbsCbsRecipient struct {
	TaxID       string
	NIF         string
	NoNifReason string
	Name        string
	Address     *IbsCbsRecipientAddress
	Phone       string
	Email       string
}

func (r IbsCbsRecipient) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.pattern("tax_id", r.TaxID, ibsCbsRecipientID, "a CPF (11 digits) or a CNPJ (14 characters)")
	w.text("nif", r.NIF, 1, 40)
	w.choice("no_nif_reason", r.NoNifReason, "0", "1", "2")
	w.required("name", r.Name, 1, 150)
	w.pattern("phone", r.Phone, digits6to20, "6 to 20 digits")
	if has(r.Email) {
		if len([]rune(strings.TrimSpace(r.Email))) > 80 {
			w.fail("email", "must be at most 80 characters")
		}
		w.pattern("email", r.Email, emailPattern, "an e-mail address")
	}
	if r.Address != nil {
		body, err := r.Address.body(w.at("address"))
		w.adopt("address", body, err)
	}
	if presentCount(has(r.TaxID), has(r.NIF), has(r.NoNifReason)) != 1 {
		w.failWhole("takes exactly one of tax_id, nif or no_nif_reason")
	}
	return w.result()
}

// RealEstateLease is gLocacao, for a lease, onerous assignment or rental of
// real estate. CoOwnershipPercent is 100 with a single lessor. The day of
// OriginalDueDate is the only part sent.
type RealEstateLease struct {
	CoOwnershipPercent    float64
	TotalAmount           float64
	UnconditionalDiscount *float64
	ConditionalDiscount   *float64
	OriginalDueDate       time.Time
}

func (l RealEstateLease) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredNumber("co_ownership_percent", l.CoOwnershipPercent, percentOnly)
	w.requiredNumber("total_amount", l.TotalAmount, moneyOnly)
	w.number("unconditional_discount", l.UnconditionalDiscount, moneyOnly)
	w.number("conditional_discount", l.ConditionalDiscount, moneyOnly)
	w.day("original_due_date", l.OriginalDueDate)
	return w.result()
}

// RealEstateBaseAdjustment is gAjusteBCLocImoveis: Type is "01" to "04" or
// "99", and Description goes with "99".
type RealEstateBaseAdjustment struct {
	Type        string
	Description string
	Amount      float64
}

func (a RealEstateBaseAdjustment) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredChoice("type", a.Type, "01", "02", "03", "04", "99")
	w.text("description", a.Description, 1, 150)
	w.requiredNumber("amount", a.Amount, moneyOnly)
	return w.result()
}

// RealEstateUnit is one property: identify it by CIB (8 characters) or by
// Address, one of the two. BaseAdjustments are 1 to 1000 and read by the
// homologation layout only.
type RealEstateUnit struct {
	PropertyRegistration string
	CIB                  string
	Address              *ServiceAddress
	BaseAdjustments      []RealEstateBaseAdjustment
}

func (u RealEstateUnit) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.text("property_registration", u.PropertyRegistration, 1, 30)
	w.pattern("cib", u.CIB, cibPattern, "8 characters")
	if u.Address != nil {
		body, err := u.Address.body(w.at("address"))
		w.adopt("address", body, err)
	}
	if len(u.BaseAdjustments) > 1000 {
		w.fail("base_adjustments", "takes 1 to 1000 entries")
	}
	if len(u.BaseAdjustments) > 0 {
		list := make([]map[string]any, 0, len(u.BaseAdjustments))
		for i, adjustment := range u.BaseAdjustments {
			body, err := adjustment.body(fmt.Sprintf("%s[%d]", w.at("base_adjustments"), i))
			if err != nil {
				w.err = firstError(w.err, err)
				break
			}
			list = append(list, body)
		}
		w.out["base_adjustments"] = list
	}
	if has(u.CIB) == (u.Address != nil) {
		w.failWhole("takes cib or address")
	}
	return w.result()
}

// RealEstate is the imovel group. CityCode and Lease are read by the
// homologation layout only; Units are 1 to 99, and the production layout
// takes exactly one.
type RealEstate struct {
	CityCode string
	Lease    *RealEstateLease
	Units    []RealEstateUnit
}

func (r RealEstate) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.pattern("city_code", r.CityCode, digits7, "7 digits")
	if r.Lease != nil {
		body, err := r.Lease.body(w.at("lease"))
		w.adopt("lease", body, err)
	}
	if len(r.Units) > 99 {
		w.fail("units", "takes 1 to 99 entries")
	}
	if len(r.Units) > 0 {
		list := make([]map[string]any, 0, len(r.Units))
		for i, unit := range r.Units {
			body, err := unit.body(fmt.Sprintf("%s[%d]", w.at("units"), i))
			if err != nil {
				w.err = firstError(w.err, err)
				break
			}
			list = append(list, body)
		}
		w.out["units"] = list
	}
	return w.result()
}

// MovableGood is one bensMoveis entry: NCM (8 digits), Description and a
// Quantity from 1 to 999.
type MovableGood struct {
	NCM         string
	Description string
	Quantity    int
}

func (g MovableGood) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("ncm", g.NCM, digits8, "8 digits")
	w.required("description", g.Description, 1, 150)
	if g.Quantity < 1 || g.Quantity > 999 {
		w.fail("quantity", "must be 1 to 999")
	} else {
		w.out["quantity"] = g.Quantity
	}
	return w.result()
}

// IbsCbsRegularTaxation is gTribRegular: the CST (3 digits) and
// Classification (6 digits) the operation would have under regular
// taxation.
type IbsCbsRegularTaxation struct {
	CST            string
	Classification string
}

func (t IbsCbsRegularTaxation) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredPattern("cst", t.CST, digits3, "3 digits")
	w.requiredPattern("classification", t.Classification, digits6, "6 digits")
	return w.result()
}

// IbsCbsDeferral is gDif: the deferred share of each tax, in percent.
type IbsCbsDeferral struct {
	StatePercent float64
	CityPercent  float64
	CbsPercent   float64
}

func (d IbsCbsDeferral) body(path string) (map[string]any, error) {
	w := newWire(path)
	w.requiredNumber("state_percent", d.StatePercent, percentOnly)
	w.requiredNumber("city_percent", d.CityPercent, percentOnly)
	w.requiredNumber("cbs_percent", d.CbsPercent, percentOnly)
	return w.result()
}

func (g IbsCbs) nfseGroups(path string) (map[string]any, error) {
	w := newWire(path)
	w.flag("personal_use", g.PersonalUse)
	if g.PresumedCredit != nil {
		w.pattern("presumed_credit", *g.PresumedCredit, digits2, "2 digits")
	}
	if len(g.ReferencedNfse) > 99 {
		w.fail("referenced_nfse", "takes 1 to 99 access keys")
	}
	if len(g.ReferencedNfse) > 0 {
		keys := make([]string, len(g.ReferencedNfse))
		for i, key := range g.ReferencedNfse {
			key = strings.TrimSpace(key)
			if !nfseKeyPattern.MatchString(key) {
				w.fail(fmt.Sprintf("referenced_nfse[%d]", i), "must be the 50-character access key of an NFS-e")
			}
			keys[i] = key
		}
		w.out["referenced_nfse"] = keys
	}
	if len(g.MovableGoods) > 1000 {
		w.fail("movable_goods", "takes 1 to 1000 entries")
	}
	if len(g.MovableGoods) > 0 {
		list := make([]map[string]any, 0, len(g.MovableGoods))
		for i, good := range g.MovableGoods {
			body, err := good.body(fmt.Sprintf("%s[%d]", w.at("movable_goods"), i))
			if err != nil {
				w.err = firstError(w.err, err)
				break
			}
			list = append(list, body)
		}
		w.out["movable_goods"] = list
	}
	groups := []struct {
		key  string
		set  bool
		body func(string) (map[string]any, error)
	}{
		{"government", g.Government != nil, func(p string) (map[string]any, error) { return g.Government.body(p) }},
		{"recipient", g.Recipient != nil, func(p string) (map[string]any, error) { return g.Recipient.body(p) }},
		{"real_estate", g.RealEstate != nil, func(p string) (map[string]any, error) { return g.RealEstate.body(p) }},
		{"regular_taxation", g.RegularTaxation != nil, func(p string) (map[string]any, error) { return g.RegularTaxation.body(p) }},
		{"deferral", g.Deferral != nil, func(p string) (map[string]any, error) { return g.Deferral.body(p) }},
	}
	for _, group := range groups {
		if group.set {
			body, err := group.body(w.at(group.key))
			w.adopt(group.key, body, err)
		}
	}
	return w.result()
}
