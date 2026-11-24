package stackin

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stackin-io/stackin-go-sdk/br"
)

func ptrOf[T any](v T) *T { return &v }

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

func serviceLine(mutate func(*br.Product)) br.Product {
	line := br.Product{Description: "Consultoria", Amount: 100, ServiceCode: ptrOf("1.07")}
	if mutate != nil {
		mutate(&line)
	}
	return line
}

func nfseRequest(details *br.NfseDetails) IssueRequest {
	return IssueRequest{
		DocumentType: NFSE,
		ClientName:   "Maria",
		TaxID:        "12345678909",
		Items:        []br.Product{serviceLine(nil)},
		Nfse:         details,
	}
}

func TestNfseGroupsGoOutAsSnakeCaseWithDecimalStrings(t *testing.T) {
	key := strings.Repeat("1", 50)
	raw, called, err := captureIssue(t, nfseRequest(&br.NfseDetails{
		ApproximateTaxes: &br.ApproximateTaxes{FederalPercent: ptrOf(13.45), StatePercent: ptrOf(0.0)},
		CompetenceDate:   day(2026, time.November, 20),
		ServiceCityCode:  "3550308",
		Iss: &br.IssDetails{
			Taxation:         "taxable",
			Rate:             ptrOf(2.5),
			Retention:        "tomador",
			Suspension:       &br.IssSuspension{Type: "judicial", ProcessNumber: strings.Repeat("7", 30)},
			MunicipalBenefit: &br.IssMunicipalBenefit{ID: strings.Repeat("3", 14), ReductionPercent: ptrOf(10.0)},
		},
		Federal: &br.FederalTaxes{
			PisCofins:              &br.PisCofins{CST: "01", Base: ptrOf(100.0), PisRate: ptrOf(0.65), WithholdingType: ptrOf(0)},
			SocialSecurityWithheld: ptrOf(1.5),
		},
		Deductions: &br.Deductions{Documents: []br.DeductionDocument{{
			Type:           2,
			NfseKey:        key,
			IssueDate:      day(2026, time.November, 1),
			TotalAmount:    50,
			DeductedAmount: 20.5,
		}}},
		Substitution:   &br.Substitution{AccessKey: key, ReasonCode: "5"},
		Construction:   &br.Construction{WorkIDType: "cib", WorkID: "12345678"},
		Event:          &br.Event{Name: "Show", StartDate: day(2026, time.November, 2), EndDate: day(2026, time.November, 3), EventID: "EV1"},
		References:     &br.References{OrderNumber: "PO-1", OrderItems: []string{"a", "b"}},
		Provider:       &br.NfseProvider{Caepf: "12345678901234", Phone: "+55 (11) 99999-0000", Email: "dps@example.com"},
		RecipientCaepf: "43210987654321",
		Intermediary: &br.NfseIntermediary{
			ForeignID: "NIF1",
			Name:      "Broker",
			Address:   &br.PersonAddress{Street: "Rua", Number: "1", Neighborhood: "Centro", CountryCode: "PT", PostalCode: "1000", City: "Lisboa", State: "Lisboa"},
		},
		ForeignTrade: &br.ForeignTrade{
			ServiceMode: 1, Relationship: ptrOf(0), CurrencyCode: "220", ForeignAmount: 10.5,
			ProviderSupport: 1, RecipientSupport: 1, TemporaryGoods: 2, ImportDeclaration: "DI1", ShareWithMdic: true,
		},
	}))
	if err != nil || !called {
		t.Fatalf("called = %v, err = %v", called, err)
	}
	got := sentFields(t, raw)["nfse"]
	want := map[string]any{
		"approximate_taxes": map[string]any{"federal_percent": "13.45", "state_percent": "0"},
		"competence_date":   "2026-11-20",
		"service_city_code": "3550308",
		"iss": map[string]any{
			"taxation":          "taxable",
			"rate":              "2.5",
			"retention":         "tomador",
			"suspension":        map[string]any{"type": "judicial", "process_number": strings.Repeat("7", 30)},
			"municipal_benefit": map[string]any{"id": strings.Repeat("3", 14), "reduction_percent": "10"},
		},
		"federal": map[string]any{
			"pis_cofins":               map[string]any{"cst": "01", "base": "100", "pis_rate": "0.65", "withholding_type": float64(0)},
			"social_security_withheld": "1.5",
		},
		"deductions": map[string]any{"documents": []any{map[string]any{
			"type": float64(2), "nfse_key": key, "issue_date": "2026-11-01",
			"total_amount": "50", "deducted_amount": "20.5",
		}}},
		"substitution":    map[string]any{"access_key": key, "reason_code": "05"},
		"construction":    map[string]any{"work_id_type": "cib", "work_id": "12345678"},
		"event":           map[string]any{"name": "Show", "start_date": "2026-11-02", "end_date": "2026-11-03", "event_id": "EV1"},
		"references":      map[string]any{"order_number": "PO-1", "order_items": []any{"a", "b"}},
		"provider":        map[string]any{"caepf": "12345678901234", "phone": "+5511999990000", "email": "dps@example.com"},
		"recipient_caepf": "43210987654321",
		"intermediary": map[string]any{
			"foreign_id": "NIF1",
			"name":       "Broker",
			"address": map[string]any{
				"street": "Rua", "number": "1", "neighborhood": "Centro",
				"country_code": "PT", "postal_code": "1000", "city": "Lisboa", "state": "Lisboa",
			},
		},
		"foreign_trade": map[string]any{
			"service_mode": float64(1), "relationship": float64(0), "currency_code": "220",
			"foreign_amount": "10.5", "provider_support": float64(1), "recipient_support": float64(1),
			"temporary_goods": float64(2), "import_declaration": "DI1", "share_with_mdic": true,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nfse = %#v\nwant %#v", got, want)
	}
}

func TestNfseIsLeftOutWhenNothingIsSet(t *testing.T) {
	for _, details := range []*br.NfseDetails{nil, {}} {
		raw, _, err := captureIssue(t, nfseRequest(details))
		if err != nil {
			t.Fatal(err)
		}
		if _, sent := sentFields(t, raw)["nfse"]; sent {
			t.Errorf("nfse sent for %+v: %s", details, raw)
		}
	}
}

func TestNfseRefusalsNeverReachTheNetwork(t *testing.T) {
	key := strings.Repeat("1", 50)
	national := br.PersonAddress{Street: "Rua", Number: "1", Neighborhood: "Centro", CityCode: "3550308", ZipCode: "01001000"}
	abroad := br.PersonAddress{Street: "Rua", Number: "1", Neighborhood: "Centro", CountryCode: "PT", PostalCode: "1000", City: "Lisboa", State: "Lisboa"}
	cases := map[string]struct {
		details br.NfseDetails
		want    string
	}{
		"approximate taxes empty": {
			br.NfseDetails{ApproximateTaxes: &br.ApproximateTaxes{}},
			"nfse.approximate_taxes needs",
		},
		"approximate taxes two shapes": {
			br.NfseDetails{ApproximateTaxes: &br.ApproximateTaxes{SimplesPercent: ptrOf(3.0), FederalAmount: ptrOf(1.0)}},
			"takes one shape",
		},
		"simples percent above cap": {
			br.NfseDetails{ApproximateTaxes: &br.ApproximateTaxes{SimplesPercent: ptrOf(100.0)}},
			"nfse.approximate_taxes.simples_percent must be",
		},
		"percent with three places": {
			br.NfseDetails{ApproximateTaxes: &br.ApproximateTaxes{FederalPercent: ptrOf(1.234)}},
			"federal_percent must be",
		},
		"iss rate zero": {
			br.NfseDetails{Iss: &br.IssDetails{Rate: ptrOf(0.0)}},
			"nfse.iss.rate must be above 0",
		},
		"iss taxation unknown": {
			br.NfseDetails{Iss: &br.IssDetails{Taxation: "free"}},
			"nfse.iss.taxation must be one of",
		},
		"immunity type": {
			br.NfseDetails{Iss: &br.IssDetails{ImmunityType: ptrOf(0)}},
			"immunity_type must be one of",
		},
		"suspension process": {
			br.NfseDetails{Iss: &br.IssDetails{Suspension: &br.IssSuspension{Type: "judicial", ProcessNumber: "12"}}},
			"process_number must be 30 digits",
		},
		"benefit both reductions": {
			br.NfseDetails{Iss: &br.IssDetails{MunicipalBenefit: &br.IssMunicipalBenefit{ID: strings.Repeat("1", 14), ReductionAmount: ptrOf(1.0), ReductionPercent: ptrOf(1.0)}}},
			"not both",
		},
		"federal empty": {
			br.NfseDetails{Federal: &br.FederalTaxes{}},
			"nfse.federal needs",
		},
		"pis cofins cst": {
			br.NfseDetails{Federal: &br.FederalTaxes{PisCofins: &br.PisCofins{CST: "10"}}},
			"pis_cofins.cst must be one of",
		},
		"withheld zero": {
			br.NfseDetails{Federal: &br.FederalTaxes{IrrfWithheld: ptrOf(0.0)}},
			"irrf_withheld must be above 0",
		},
		"service place both": {
			br.NfseDetails{ServiceCityCode: "3550308", ServiceCountryCode: "PT"},
			"not both",
		},
		"service country BR": {
			br.NfseDetails{ServiceCountryCode: "BR"},
			"cannot be BR",
		},
		"service city code": {
			br.NfseDetails{ServiceCityCode: "355"},
			"service_city_code must be",
		},
		"competence year": {
			br.NfseDetails{CompetenceDate: day(1999, time.January, 1)},
			"competence_date must be a date from 2000 to 2099",
		},
		"construction empty": {
			br.NfseDetails{Construction: &br.Construction{}},
			"needs work_id",
		},
		"construction both": {
			br.NfseDetails{Construction: &br.Construction{WorkIDType: "cno", WorkID: "1", Address: &br.ServiceAddress{ZipCode: "01001000", Street: "R", Number: "1", Neighborhood: "C"}}},
			"not both",
		},
		"construction cib length": {
			br.NfseDetails{Construction: &br.Construction{WorkIDType: "cib", WorkID: "123"}},
			"exactly 8 characters",
		},
		"construction id without type": {
			br.NfseDetails{Construction: &br.Construction{WorkID: "123"}},
			"go together",
		},
		"construction address abroad service": {
			br.NfseDetails{ServiceCountryCode: "PT", Construction: &br.Construction{Address: &br.ServiceAddress{ZipCode: "01001000", Street: "R", Number: "1", Neighborhood: "C"}}},
			"construction.address is an address in Brazil",
		},
		"event place": {
			br.NfseDetails{Event: &br.Event{Name: "Show", StartDate: day(2026, 1, 1), EndDate: day(2026, 1, 2)}},
			"takes event_id or address",
		},
		"event dates": {
			br.NfseDetails{Event: &br.Event{Name: "Show", EventID: "E"}},
			"start_date is required",
		},
		"references too many items": {
			br.NfseDetails{References: &br.References{OrderItems: make([]string, 100)}},
			"at most 99",
		},
		"deductions empty": {
			br.NfseDetails{Deductions: &br.Deductions{}},
			"needs percent, amount or documents",
		},
		"deductions both": {
			br.NfseDetails{Deductions: &br.Deductions{Percent: ptrOf(1.0), Amount: ptrOf(1.0)}},
			"not both",
		},
		"deduction document type": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{Type: 10, NfseKey: key, IssueDate: day(2026, 1, 1), TotalAmount: 1, DeductedAmount: 1}}}},
			"documents[0].type must be one of",
		},
		"deduction document identification": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{Type: 2, IssueDate: day(2026, 1, 1), TotalAmount: 1, DeductedAmount: 1}}}},
			"exactly one of nfse_key",
		},
		"deduction supplier ids": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{
				Type: 7, OtherDocument: &br.OtherDocument{Number: "1"}, IssueDate: day(2026, 1, 1), TotalAmount: 1, DeductedAmount: 1,
				Supplier: &br.DeductionSupplier{Name: "S", Address: &abroad},
			}}}},
			"supplier takes tax_id",
		},
		"deduction supplier tax id with abroad address": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{
				Type: 7, OtherDocument: &br.OtherDocument{Number: "1"}, IssueDate: day(2026, 1, 1), TotalAmount: 1, DeductedAmount: 1,
				Supplier: &br.DeductionSupplier{TaxID: "12345678909", Name: "S", Address: &abroad},
			}}}},
			"supplier.address must be in Brazil",
		},
		"substitution key": {
			br.NfseDetails{Substitution: &br.Substitution{AccessKey: "12", ReasonCode: "01"}},
			"access_key must be",
		},
		"substitution other without reason": {
			br.NfseDetails{Substitution: &br.Substitution{AccessKey: key, ReasonCode: "99"}},
			"reason is required with reason_code 99",
		},
		"substitution reason code": {
			br.NfseDetails{Substitution: &br.Substitution{AccessKey: key, ReasonCode: "07"}},
			"reason_code must be one of",
		},
		"provider caepf": {
			br.NfseDetails{Provider: &br.NfseProvider{Caepf: "123"}},
			"caepf must be 14 digits",
		},
		"provider phone": {
			br.NfseDetails{Provider: &br.NfseProvider{Phone: "12"}},
			"phone must be 8 to 15 digits",
		},
		"provider email": {
			br.NfseDetails{Provider: &br.NfseProvider{Email: "nope"}},
			"email must be an e-mail address",
		},
		"recipient caepf": {
			br.NfseDetails{RecipientCaepf: "1234"},
			"recipient_caepf must be 14 digits",
		},
		"other dfe without description": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{Type: 99, Description: "x", OtherDfe: &br.OtherDfe{Key: "K"}, IssueDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), TotalAmount: 5, DeductedAmount: 5}}}},
			"description is required",
		},
		"other dfe and nfse key": {
			br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{Type: 2, NfseKey: key, OtherDfe: &br.OtherDfe{Key: "K", Description: "D"}, IssueDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), TotalAmount: 5, DeductedAmount: 5}}}},
			"exactly one of nfse_key",
		},
		"intermediary two ids": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{TaxID: "12345678909", ForeignID: "N", Name: "I", Address: &abroad}},
			"exactly one of tax_id",
		},
		"intermediary foreign id with national address": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{ForeignID: "N", Name: "I", Address: &national}},
			"must be abroad",
		},
		"intermediary tax id with abroad address": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{TaxID: "12345678909", Name: "I", Address: &abroad}},
			"not tax_id",
		},
		"intermediary missing reason zero": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{ForeignIDMissingReason: ptrOf(0), Name: "I", Address: &abroad}},
			"foreign_id_missing_reason must be one of",
		},
		"address both kinds": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{TaxID: "12345678909", Name: "I", Address: &br.PersonAddress{Street: "R", Number: "1", Neighborhood: "C", CityCode: "3550308", ZipCode: "01001000", CountryCode: "PT"}}},
			"takes city_code and zip_code",
		},
		"address abroad BR": {
			br.NfseDetails{Intermediary: &br.NfseIntermediary{ForeignID: "N", Name: "I", Address: &br.PersonAddress{Street: "R", Number: "1", Neighborhood: "C", CountryCode: "BR", PostalCode: "1", City: "C", State: "S"}}},
			"cannot be BR",
		},
		"foreign trade zero code": {
			br.NfseDetails{ForeignTrade: &br.ForeignTrade{Relationship: ptrOf(0), CurrencyCode: "220", ForeignAmount: 1, ProviderSupport: 1, RecipientSupport: 1, TemporaryGoods: 1}},
			"service_mode must be one of",
		},
		"foreign trade relationship missing": {
			br.NfseDetails{ForeignTrade: &br.ForeignTrade{ServiceMode: 1, CurrencyCode: "220", ForeignAmount: 1, ProviderSupport: 1, RecipientSupport: 1, TemporaryGoods: 1}},
			"relationship is required",
		},
		"foreign trade declaration": {
			br.NfseDetails{ForeignTrade: &br.ForeignTrade{ServiceMode: 1, Relationship: ptrOf(1), CurrencyCode: "220", ForeignAmount: 1, ProviderSupport: 1, RecipientSupport: 1, TemporaryGoods: 3}},
			"needs export_registration",
		},
		"foreign trade recipient support": {
			br.NfseDetails{ForeignTrade: &br.ForeignTrade{ServiceMode: 1, Relationship: ptrOf(1), CurrencyCode: "220", ForeignAmount: 1, ProviderSupport: 1, RecipientSupport: 27, TemporaryGoods: 1}},
			"recipient_support must be one of",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			details := tc.details
			expectRefusal(t, nfseRequest(&details), tc.want)
		})
	}
}

func TestRecipientFieldsGoOut(t *testing.T) {
	req := nfseRequest(nil)
	req.RecipientMunicipalRegistration = "12.345-6"
	req.ForeignIDMissingReason = ptrOf(2)
	req.RecipientAddress = &Address{
		Street: "Fifth Avenue", Number: "350", Neighborhood: "Midtown", City: "New York", State: "EX",
		CountryISO2: "US", PostalCode: "10118", Province: "NY", Complement: "Suite 5",
	}
	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, raw)
	if body["recipient_municipal_registration"] != "123456" || body["foreign_id_missing_reason"] != float64(2) {
		t.Errorf("body = %s", raw)
	}
	address := body["recipient_address"].(map[string]any)
	for key, want := range map[string]string{"country_iso2": "US", "postal_code": "10118", "province": "NY", "complement": "Suite 5"} {
		if address[key] != want {
			t.Errorf("recipient_address.%s = %v, want %s", key, address[key], want)
		}
	}
}

func TestRecipientFieldsAreLeftOutWhenEmpty(t *testing.T) {
	raw, _, err := captureIssue(t, nfseRequest(nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"recipient_municipal_registration", "foreign_id_missing_reason", "nfse"} {
		if _, sent := sentFields(t, raw)[key]; sent {
			t.Errorf("%s sent: %s", key, raw)
		}
	}
}

func TestRecipientRefusals(t *testing.T) {
	registration := nfseRequest(nil)
	registration.RecipientMunicipalRegistration = "1234567890123456"
	expectRefusal(t, registration, "recipient_municipal_registration takes up to 15")

	reason := nfseRequest(nil)
	reason.ForeignIDMissingReason = ptrOf(0)
	expectRefusal(t, reason, "foreign_id_missing_reason takes 1")

	iso := nfseRequest(nil)
	iso.RecipientAddress = &Address{Street: "Rua", Number: "1", Neighborhood: "Centro", City: "SP", State: "SP", ZipCode: "01001000", CountryISO2: "usa"}
	expectRefusal(t, iso, "recipient_address.country_iso2")

	postal := nfseRequest(nil)
	postal.RecipientAddress = &Address{Street: "Rua", Number: "1", Neighborhood: "Centro", City: "SP", State: "SP", ZipCode: "01001000", PostalCode: strings.Repeat("1", 12)}
	expectRefusal(t, postal, "recipient_address.postal_code")
}

func TestServiceLineFieldsGoOut(t *testing.T) {
	req := nfseRequest(nil)
	req.Items = []br.Product{serviceLine(func(p *br.Product) {
		p.NBS = "115021000"
		p.MunicipalServiceCode = "001"
		p.InternalCode = "SRV01"
		p.ConditionalDiscount = ptrOf(5.5)
	})}
	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	item := sentFields(t, raw)["items"].([]any)[0].(map[string]any)
	want := map[string]any{"nbs": "115021000", "municipal_service_code": "001", "internal_code": "SRV01", "conditional_discount": "5.5"}
	for key, value := range want {
		if item[key] != value {
			t.Errorf("items[0].%s = %v, want %v", key, item[key], value)
		}
	}
}

func TestServiceLineRefusals(t *testing.T) {
	cases := map[string]struct {
		mutate func(*br.Product)
		want   string
	}{
		"municipal code":  {func(p *br.Product) { p.MunicipalServiceCode = "12" }, "items[0].municipal_service_code must be 3 digits"},
		"internal code":   {func(p *br.Product) { p.InternalCode = "ab-1" }, "items[0].internal_code"},
		"discount zero":   {func(p *br.Product) { p.ConditionalDiscount = ptrOf(0.0) }, "items[0].conditional_discount must be above 0"},
		"discount places": {func(p *br.Product) { p.ConditionalDiscount = ptrOf(1.234) }, "items[0].conditional_discount"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := nfseRequest(nil)
			req.Items = []br.Product{serviceLine(tc.mutate)}
			expectRefusal(t, req, tc.want)
		})
	}
}

func TestIbsCbsNfseGroupsGoOut(t *testing.T) {
	req := nfseRequest(nil)
	req.Items = []br.Product{serviceLine(func(p *br.Product) {
		p.NBS = "115021000"
		p.IbsCbs = &br.IbsCbs{
			CST: "000", Classification: "000001", OperationIndicator: ptrOf("100301"),
			PersonalUse:    ptrOf(false),
			Government:     &br.IbsCbsGovernment{OperationType: "2", EntityType: "4"},
			ReferencedNfse: []string{strings.Repeat("2", 50)},
			Recipient: &br.IbsCbsRecipient{
				NIF: "N1", Name: "Dest",
				Address: &br.IbsCbsRecipientAddress{Street: "R", Number: "1", Neighborhood: "C", CountryCode: "PT", PostalCode: "1", City: "L", Region: "L"},
			},
			RealEstate: &br.RealEstate{
				CityCode: "3550308",
				Lease:    &br.RealEstateLease{CoOwnershipPercent: 100, TotalAmount: 1500, OriginalDueDate: day(2026, 12, 5)},
				Units:    []br.RealEstateUnit{{CIB: "ABCD1234", BaseAdjustments: []br.RealEstateBaseAdjustment{{Type: "99", Description: "x", Amount: 10}}}},
			},
			MovableGoods:    []br.MovableGood{{NCM: "84713012", Description: "Notebook", Quantity: 2}},
			RegularTaxation: &br.IbsCbsRegularTaxation{CST: "000", Classification: "000001"},
			Deferral:        &br.IbsCbsDeferral{StatePercent: 0, CityPercent: 10, CbsPercent: 50},
			PresumedCredit:  ptrOf("01"),
		}
	})}
	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	group := sentFields(t, raw)["items"].([]any)[0].(map[string]any)["product"].(map[string]any)["br"].(map[string]any)["ibs_cbs"].(map[string]any)
	want := map[string]any{
		"cst": "000", "classification": "000001", "operation_indicator": "100301",
		"personal_use":    false,
		"government":      map[string]any{"operation_type": "2", "entity_type": "4"},
		"referenced_nfse": []any{strings.Repeat("2", 50)},
		"recipient": map[string]any{
			"nif": "N1", "name": "Dest",
			"address": map[string]any{"street": "R", "number": "1", "neighborhood": "C", "country_code": "PT", "postal_code": "1", "city": "L", "region": "L"},
		},
		"real_estate": map[string]any{
			"city_code": "3550308",
			"lease":     map[string]any{"co_ownership_percent": float64(100), "total_amount": float64(1500), "original_due_date": "2026-12-05"},
			"units": []any{map[string]any{
				"cib":              "ABCD1234",
				"base_adjustments": []any{map[string]any{"type": "99", "description": "x", "amount": float64(10)}},
			}},
		},
		"movable_goods":    []any{map[string]any{"ncm": "84713012", "description": "Notebook", "quantity": float64(2)}},
		"regular_taxation": map[string]any{"cst": "000", "classification": "000001"},
		"deferral":         map[string]any{"state_percent": float64(0), "city_percent": float64(10), "cbs_percent": float64(50)},
		"presumed_credit":  "01",
	}
	if !reflect.DeepEqual(group, want) {
		t.Errorf("ibs_cbs = %#v\nwant %#v", group, want)
	}
}

func TestIbsCbsNfseRefusals(t *testing.T) {
	cases := map[string]struct {
		group br.IbsCbs
		want  string
	}{
		"government empty":  {br.IbsCbs{Government: &br.IbsCbsGovernment{}}, "ibs_cbs.government needs"},
		"government type":   {br.IbsCbs{Government: &br.IbsCbsGovernment{OperationType: "6"}}, "government.operation_type must be one of"},
		"referenced key":    {br.IbsCbs{ReferencedNfse: []string{"1"}}, "ibs_cbs.referenced_nfse[0] must be"},
		"recipient two ids": {br.IbsCbs{Recipient: &br.IbsCbsRecipient{TaxID: "12345678909", NIF: "N", Name: "D"}}, "exactly one of tax_id"},
		"recipient address": {br.IbsCbs{Recipient: &br.IbsCbsRecipient{NoNifReason: "1", Name: "D", Address: &br.IbsCbsRecipientAddress{Street: "R", Number: "1", Neighborhood: "C"}}}, "recipient.address in Brazil needs city_code and zip_code"},
		"unit place":        {br.IbsCbs{RealEstate: &br.RealEstate{Units: []br.RealEstateUnit{{}}}}, "units[0] takes cib or address"},
		"adjustment type":   {br.IbsCbs{RealEstate: &br.RealEstate{Units: []br.RealEstateUnit{{CIB: "ABCD1234", BaseAdjustments: []br.RealEstateBaseAdjustment{{Type: "05"}}}}}}, "base_adjustments[0].type must be one of"},
		"lease percent":     {br.IbsCbs{RealEstate: &br.RealEstate{Lease: &br.RealEstateLease{CoOwnershipPercent: 101}}}, "lease.co_ownership_percent must be"},
		"good quantity":     {br.IbsCbs{MovableGoods: []br.MovableGood{{NCM: "84713012", Description: "N", Quantity: 0}}}, "movable_goods[0].quantity must be 1 to 999"},
		"regular cst":       {br.IbsCbs{RegularTaxation: &br.IbsCbsRegularTaxation{CST: "00", Classification: "000001"}}, "ibs_cbs.regular_taxation.cst must be 3 digits"},
		"deferral percent":  {br.IbsCbs{Deferral: &br.IbsCbsDeferral{StatePercent: 101}}, "ibs_cbs.deferral.state_percent must be"},
		"presumed credit":   {br.IbsCbs{PresumedCredit: ptrOf("1")}, "ibs_cbs.presumed_credit must be 2 digits"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			group := tc.group
			group.CST, group.Classification = "000", "000001"
			req := nfseRequest(nil)
			req.Items = []br.Product{serviceLine(func(p *br.Product) { p.IbsCbs = &group })}
			expectRefusal(t, req, tc.want)
		})
	}
}

func TestCompanyUpdateSendsNfseSettings(t *testing.T) {
	url, got := routeServer(t, http.StatusOK, `{}`)

	_, err := NewCompany(WithBaseURL(url), WithAPIKey("k")).Update(CompanyUpdate{
		FiscalFields: &FiscalFields{
			ApproximateTaxSimplesPercent:   ptrOf("4,5"),
			ApproximateTaxFederalPercent:   ptrOf(""),
			ApproximateTaxStatePercent:     ptrOf("0"),
			ApproximateTaxMunicipalPercent: ptrOf("1.25"),
			IssRate:                        ptrOf("2"),
			MunicipalRegistration:          ptrOf("12.345-6"),
			SpecialTaxRegime:               ptrOf("0"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"fiscal_fields": map[string]any{
		"approximate_tax_simples_percent":   "4.5",
		"approximate_tax_federal_percent":   "",
		"approximate_tax_state_percent":     "0",
		"approximate_tax_municipal_percent": "1.25",
		"iss_rate":                          "2",
		"municipal_registration":            "123456",
		"special_tax_regime":                "0",
	}}
	if !reflect.DeepEqual(got.body, want) {
		t.Errorf("body = %v, want %v", got.body, want)
	}
}

func TestCompanyUpdateRefusesBadNfseSettings(t *testing.T) {
	cases := map[string]FiscalFields{
		"iss_rate":                        {IssRate: ptrOf("0")},
		"iss_rate above":                  {IssRate: ptrOf("5.01")},
		"approximate_tax_simples_percent": {ApproximateTaxSimplesPercent: ptrOf("100")},
		"approximate_tax_state_percent":   {ApproximateTaxStatePercent: ptrOf("abc")},
		"municipal_registration":          {MunicipalRegistration: ptrOf("1234567890123456")},
		"special_tax_regime":              {SpecialTaxRegime: ptrOf("7")},
	}
	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			fields := fields
			_, err := NewCompany(WithBaseURL("http://127.0.0.1:1"), WithAPIKey("k")).Update(CompanyUpdate{FiscalFields: &fields})
			var invErr *InvoiceError
			if !errors.As(err, &invErr) || !strings.Contains(invErr.Message, strings.Fields(name)[0]) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestNBSSuggestionsReadsTheTypedAnswer(t *testing.T) {
	client, seen := newReference(t, map[string]any{
		"service_code": "010700",
		"suggestions": []any{map[string]any{
			"nbs": "115021000", "description": "Suporte técnico", "operation_indicator": "100301",
			"ibs_cbs_classification": "000001", "classification_name": nil,
		}},
		"source": map[string]any{"path": "data/nfse/anexo-viii.xlsx", "date": "2026-10-20"},
	})

	got, err := client.NBSSuggestions("1.07")
	if err != nil {
		t.Fatal(err)
	}
	if request := seen(); request.Method != http.MethodGet || request.URL.Path != "/api/v1/service-codes/1.07/nbs" {
		t.Errorf("request = %s %s", request.Method, request.URL.Path)
	}
	if got.ServiceCode != "010700" || len(got.Suggestions) != 1 || got.Source.Path != "data/nfse/anexo-viii.xlsx" || *got.Source.Date != "2026-10-20" {
		t.Fatalf("got = %+v", got)
	}
	row := got.Suggestions[0]
	if row.NBS != "115021000" || *row.OperationIndicator != "100301" || *row.IbsCbsClassification != "000001" || row.ClassificationName != nil {
		t.Errorf("row = %+v", row)
	}
}

func TestNBSSuggestionsAnswersEmptyListNotNil(t *testing.T) {
	client, _ := newReference(t, map[string]any{"service_code": "990101", "suggestions": nil, "source": map[string]any{"path": "p", "date": nil}})

	got, err := client.NBSSuggestions("99.01.01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Suggestions == nil || len(got.Suggestions) != 0 || got.Source.Date != nil {
		t.Errorf("got = %+v", got)
	}
}

func TestNBSSuggestionsRefusesUnusableCodes(t *testing.T) {
	client := NewFiscalReference(WithBaseURL("http://127.0.0.1:1"), WithAPIKey("k"))
	for _, code := range []string{"", "..", "1.07.00.00.00.0"} {
		var invErr *InvoiceError
		if _, err := client.NBSSuggestions(code); !errors.As(err, &invErr) {
			t.Errorf("code %q: err = %v", code, err)
		}
	}
}

func TestNfseDeductionOtherDfeIsSent(t *testing.T) {
	raw, _, err := captureIssue(t, nfseRequest(&br.NfseDetails{Deductions: &br.Deductions{Documents: []br.DeductionDocument{{
		Type:           99,
		Description:    "outros",
		OtherDfe:       &br.OtherDfe{Key: "ABC123", Description: "Fatura de energia"},
		IssueDate:      day(2026, time.November, 1),
		TotalAmount:    50,
		DeductedAmount: 20,
	}}}}))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	documents := sentFields(t, raw)["nfse"].(map[string]any)["deductions"].(map[string]any)["documents"].([]any)
	got := documents[0].(map[string]any)["other_dfe"]
	want := map[string]any{"key": "ABC123", "description": "Fatura de energia"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("other_dfe = %#v, want %#v", got, want)
	}
}
