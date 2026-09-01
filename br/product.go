package br

type PresumedCredit struct {
	Code       string  `json:"code"`
	Percentage float64 `json:"percentage"`
	Amount     float64 `json:"amount"`
}

type Product struct {
	Description       string
	Amount            float64
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
	for k, v := range p.ExtraGroups {
		br[k] = v
	}

	if len(br) > 0 {
		data["br"] = br
	}

	result := map[string]any{
		"description":  p.Description,
		"amount":       p.Amount,
		"product":      data,
		"tax_retained": p.TaxRetained,
	}
	setIfNotNil(result, "service_code", p.ServiceCode)
	setIfNotNil(result, "discount", p.ServiceDiscount)
	setIfNotNil(result, "observations", p.Observations)
	return result
}

func setIfNotNil[T any](m map[string]any, key string, v *T) {
	if v != nil {
		m[key] = *v
	}
}
