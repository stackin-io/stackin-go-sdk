package br

import (
	"reflect"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

func TestProductToDictMinimal(t *testing.T) {
	p := Product{Description: "Servico basico", Amount: 100.0}
	data := p.ToDict()

	want := map[string]any{
		"description":  "Servico basico",
		"amount":       100.0,
		"product":      map[string]any{"unit": "UN", "quantity": 1.0, "used_movable_asset": false},
		"tax_retained": false,
	}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("ToDict() = %#v, want %#v", data, want)
	}
}

func TestProductToDictDefaultsUnitAndQuantity(t *testing.T) {
	p := Product{Description: "Produto", Amount: 1.0}
	data := p.ToDict()
	product := data["product"].(map[string]any)

	if product["unit"] != "UN" {
		t.Errorf("unit = %v, want UN", product["unit"])
	}
	if product["quantity"] != 1.0 {
		t.Errorf("quantity = %v, want 1.0", product["quantity"])
	}
}

func TestProductToDictKeepsExplicitUnitAndQuantity(t *testing.T) {
	p := Product{Description: "Produto", Amount: 1.0, Unit: "CX", Quantity: 20}
	data := p.ToDict()
	product := data["product"].(map[string]any)

	if product["unit"] != "CX" {
		t.Errorf("unit = %v, want CX", product["unit"])
	}
	if product["quantity"] != 20.0 {
		t.Errorf("quantity = %v, want 20", product["quantity"])
	}
}

func TestProductToDictNestsBRFields(t *testing.T) {
	p := Product{
		Description: "Produto",
		Amount:      50.0,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
		CEST:        ptr("0300700"),
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)

	if brData["ncm"] != "84713012" {
		t.Errorf("ncm = %v, want 84713012", brData["ncm"])
	}
	if brData["cfop"] != "5102" {
		t.Errorf("cfop = %v, want 5102", brData["cfop"])
	}
	if brData["cest"] != "0300700" {
		t.Errorf("cest = %v, want 0300700", brData["cest"])
	}
}

func TestProductToDictOmitsBRWhenEmpty(t *testing.T) {
	p := Product{Description: "Servico", Amount: 10.0}
	data := p.ToDict()
	product := data["product"].(map[string]any)

	if _, ok := product["br"]; ok {
		t.Error("br key present, want absent for a service with no BR fields")
	}
}

func TestProductToDictNVECodes(t *testing.T) {
	p := Product{
		Description: "Produto",
		Amount:      1.0,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
		NVECodes:    []string{"NV0001", "NV0002"},
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)

	got := brData["nve_codes"].([]string)
	if !reflect.DeepEqual(got, []string{"NV0001", "NV0002"}) {
		t.Errorf("nve_codes = %v, want [NV0001 NV0002]", got)
	}
}

func TestProductToDictPresumedCredits(t *testing.T) {
	p := Product{
		Description: "Produto",
		Amount:      1.0,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
		PresumedCredits: []PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 2.40},
		},
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)

	credits := brData["presumed_credits"].([]PresumedCredit)
	if len(credits) != 1 || credits[0].Code != "PR820001" {
		t.Errorf("presumed_credits = %v, want one credit with code PR820001", credits)
	}
}

func TestProductToDictExtraGroupsMergedIntoBR(t *testing.T) {
	p := Product{
		Description: "Produto",
		Amount:      1.0,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
		ExtraGroups: map[string]any{"custom_field": "value"},
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)

	if brData["custom_field"] != "value" {
		t.Errorf("custom_field = %v, want value", brData["custom_field"])
	}
}

func TestProductToDictWithTax(t *testing.T) {
	p := Product{
		Description: "Produto",
		Amount:      1.0,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
		Tax:         &Tax{Icms: IcmsSn102{Orig: ptr("0"), CSOSN: "102"}},
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)
	taxData := brData["tax"].(map[string]any)

	icms := taxData["ICMS"].(map[string]any)
	if _, ok := icms["ICMSSN102"]; !ok {
		t.Errorf("ICMS = %v, want ICMSSN102 key", icms)
	}
}

func TestProductToDictNFSeFields(t *testing.T) {
	p := Product{
		Description:     "Consultoria",
		Amount:          1500.0,
		ServiceCode:     ptr("1.06"),
		ServiceDiscount: ptr(50.0),
		TaxRetained:     true,
		Observations:    ptr("Nota de teste"),
	}
	data := p.ToDict()

	if data["service_code"] != "1.06" {
		t.Errorf("service_code = %v, want 1.06", data["service_code"])
	}
	if data["discount"] != 50.0 {
		t.Errorf("discount = %v, want 50.0", data["discount"])
	}
	if data["tax_retained"] != true {
		t.Errorf("tax_retained = %v, want true", data["tax_retained"])
	}
	if data["observations"] != "Nota de teste" {
		t.Errorf("observations = %v, want %q", data["observations"], "Nota de teste")
	}
}

func TestProductToDictExtraExpenses(t *testing.T) {
	p := Product{
		Description:       "Produto",
		Amount:            10.0,
		Barcode:           ptr("7891000100103"),
		Freight:           ptr(15.0),
		Insurance:         ptr(5.0),
		Discount:          ptr(10.0),
		OtherExpenses:     ptr(3.5),
		UsedMovableAsset:  true,
		PurchaseOrder:     ptr("PC-1"),
		PurchaseOrderItem: ptr("1"),
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)

	if product["barcode"] != "7891000100103" {
		t.Errorf("barcode = %v", product["barcode"])
	}
	if product["freight"] != 15.0 {
		t.Errorf("freight = %v", product["freight"])
	}
	if product["insurance"] != 5.0 {
		t.Errorf("insurance = %v", product["insurance"])
	}
	if product["discount"] != 10.0 {
		t.Errorf("discount = %v", product["discount"])
	}
	if product["other_expenses"] != 3.5 {
		t.Errorf("other_expenses = %v", product["other_expenses"])
	}
	if product["used_movable_asset"] != true {
		t.Errorf("used_movable_asset = %v", product["used_movable_asset"])
	}
	if product["purchase_order"] != "PC-1" {
		t.Errorf("purchase_order = %v", product["purchase_order"])
	}
	if product["purchase_order_item"] != "1" {
		t.Errorf("purchase_order_item = %v", product["purchase_order_item"])
	}
}

func TestProductToDictRemainingBRFields(t *testing.T) {
	p := Product{
		Description:                "Produto",
		Amount:                     1.0,
		NCM:                        ptr("84713012"),
		CFOP:                       ptr("5102"),
		IndEscala:                  ptr("N"),
		ManufacturerCNPJ:           ptr("12345678000195"),
		TaxBenefitCode:             ptr("PR820001"),
		ExTipi:                     ptr("01"),
		ImportContentControlNumber: ptr("550E8400-E29B-41D4-A716-446655440000"),
		RecopiNumber:               ptr("00000000000012345678"),
	}
	data := p.ToDict()
	product := data["product"].(map[string]any)
	brData := product["br"].(map[string]any)

	if brData["ind_escala"] != "N" {
		t.Errorf("ind_escala = %v", brData["ind_escala"])
	}
	if brData["manufacturer_cnpj"] != "12345678000195" {
		t.Errorf("manufacturer_cnpj = %v", brData["manufacturer_cnpj"])
	}
	if brData["tax_benefit_code"] != "PR820001" {
		t.Errorf("tax_benefit_code = %v", brData["tax_benefit_code"])
	}
	if brData["ex_tipi"] != "01" {
		t.Errorf("ex_tipi = %v", brData["ex_tipi"])
	}
	if brData["import_content_control_number"] != "550E8400-E29B-41D4-A716-446655440000" {
		t.Errorf("import_content_control_number = %v", brData["import_content_control_number"])
	}
	if brData["recopi_number"] != "00000000000012345678" {
		t.Errorf("recopi_number = %v", brData["recopi_number"])
	}
}
