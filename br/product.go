package br

import "strconv"

type PresumedCredit struct {
	Code       string  `json:"code"`
	Percentage float64 `json:"percentage"`
	Amount     float64 `json:"amount"`
}

type IbsCbs struct {
	CST            string
	Classification string
	Base           *float64
	RateState      float64
	RateCity       float64
	RateFederal    float64
}

func (g IbsCbs) toMap() map[string]any {
	data := map[string]any{
		"cst":            g.CST,
		"classification": g.Classification,
		"rate_state":     g.RateState,
		"rate_city":      g.RateCity,
		"rate_federal":   g.RateFederal,
	}
	setIfNotNil(data, "base", g.Base)
	return data
}

type Product struct {
	Description       string
	Amount            float64
	UnitPrice         *float64
	Unit              string
	Quantity          float64
	Barcode           *string
	Freight           *float64
	Insurance         *float64
	Discount          *float64
	OtherExpenses     *float64
	UsedMovableAsset  bool
	PurchaseOrder     *string
	PurchaseOrderItem *string

	NCM                        *string
	CFOP                       *string
	CEST                       *string
	NVECodes                   []string
	IndEscala                  *string
	ManufacturerCNPJ           *string
	TaxBenefitCode             *string
	PresumedCredits            []PresumedCredit
	ExTipi                     *string
	ImportContentControlNumber *string
	RecopiNumber               *string
	ExtraGroups                map[string]any
	Tax                        *Tax
	IbsCbs                     *IbsCbs

	ServiceCode     *string  // LC 116/2003 item.subitem, nfse only
	ServiceDiscount *float64 // unconditional discount, nfse only
	TaxRetained     bool     // ISSQN retained by the tomador, nfse only
	Observations    *string  // nfse only
}

func (p Product) ToDict() map[string]any {
	unit := p.Unit
	if unit == "" {
		unit = "UN"
	}
	quantity := p.Quantity
	if quantity == 0 {
		quantity = 1.0
	}

	data := map[string]any{
		"unit":               unit,
		"quantity":           quantity,
		"used_movable_asset": p.UsedMovableAsset,
	}
	setIfNotNil(data, "barcode", p.Barcode)
	setIfNotNil(data, "freight", p.Freight)
	setIfNotNil(data, "insurance", p.Insurance)
	setIfNotNil(data, "discount", p.Discount)
	setIfNotNil(data, "other_expenses", p.OtherExpenses)
	setIfNotNil(data, "purchase_order", p.PurchaseOrder)
	setIfNotNil(data, "purchase_order_item", p.PurchaseOrderItem)

	br := map[string]any{}
	setIfNotNil(br, "ncm", p.NCM)
	setIfNotNil(br, "cfop", p.CFOP)
	setIfNotNil(br, "cest", p.CEST)
	if len(p.NVECodes) > 0 {
		br["nve_codes"] = p.NVECodes
	}
	setIfNotNil(br, "ind_escala", p.IndEscala)
	setIfNotNil(br, "manufacturer_cnpj", p.ManufacturerCNPJ)
	setIfNotNil(br, "tax_benefit_code", p.TaxBenefitCode)
	if len(p.PresumedCredits) > 0 {
		br["presumed_credits"] = p.PresumedCredits
	}
	setIfNotNil(br, "ex_tipi", p.ExTipi)
	setIfNotNil(br, "import_content_control_number", p.ImportContentControlNumber)
	setIfNotNil(br, "recopi_number", p.RecopiNumber)
	if p.Tax != nil {
		br["tax"] = p.Tax.ToDict()
	}
	if p.IbsCbs != nil {
		br["ibs_cbs"] = p.IbsCbs.toMap()
	}
	for k, v := range p.ExtraGroups {
		br[k] = v
	}

	if len(br) > 0 {
		data["br"] = br
	}

	result := map[string]any{
		"description":  p.Description,
		"product":      data,
		"tax_retained": p.TaxRetained,
	}
	if p.Amount != 0 {
		result["amount"] = decimalString(p.Amount)
	}
	if p.UnitPrice != nil {
		result["unit_price"] = decimalString(*p.UnitPrice)
	}
	setIfNotNil(result, "service_code", p.ServiceCode)
	setIfNotNil(result, "discount", p.ServiceDiscount)
	setIfNotNil(result, "observations", p.Observations)
	return result
}

func decimalString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func setIfNotNil[T any](m map[string]any, key string, v *T) {
	if v != nil {
		m[key] = *v
	}
}
