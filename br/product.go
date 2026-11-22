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
	// ReductionIbs is the IBS pRedAliq (state and municipality), in percent:
	// the cClassTrib's pRedIBS. Sent only when set.
	ReductionIbs *float64
	// ReductionCbs is the CBS pRedAliq, in percent: the cClassTrib's pRedCBS.
	// Sent only when set.
	ReductionCbs *float64
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
	setIfNotNil(data, "reduction_ibs", g.ReductionIbs)
	setIfNotNil(data, "reduction_cbs", g.ReductionCbs)
	return data
}

// Fuel is the NF-e fuel group (comb), needed for a good on a fuel CFOP.
// AnpCode (cProdANP, 9 digits), AnpDescription (descANP) and ConsumerState
// (UFCons) are required; the others are sent only when set. It goes on the
// wire under the item's "fuel" key and never together with
// ExtraGroups["comb"].
type Fuel struct {
	AnpCode                   string
	AnpDescription            string
	ConsumerState             string
	GlpPercent                *float64
	NaturalGasNationalPercent *float64
	NaturalGasImportedPercent *float64
	StartingValue             *float64
	Codif                     *string
	TemperatureQuantity       *float64
	BiodieselPercent          *float64
}

func (f Fuel) toMap() map[string]any {
	data := map[string]any{
		"anp_code":        f.AnpCode,
		"anp_description": f.AnpDescription,
		"consumer_state":  f.ConsumerState,
	}
	setIfNotNil(data, "glp_percent", f.GlpPercent)
	setIfNotNil(data, "natural_gas_national_percent", f.NaturalGasNationalPercent)
	setIfNotNil(data, "natural_gas_imported_percent", f.NaturalGasImportedPercent)
	setIfNotNil(data, "starting_value", f.StartingValue)
	setIfNotNil(data, "codif", f.Codif)
	setIfNotNil(data, "temperature_quantity", f.TemperatureQuantity)
	setIfNotNil(data, "biodiesel_percent", f.BiodieselPercent)
	return data
}

type Product struct {
	// ItemID names a catalog item of the company (see stackin.Items). With
	// it, Description is optional and the item fills whatever the line
	// leaves out: Unit, Barcode, NCM, CFOP, CEST, Tax, ServiceCode,
	// TaxRetained and, when neither Amount nor UnitPrice is set, the price.
	// A field set here wins. Unit is then not defaulted to UN and
	// TaxRetained is sent only when true, so the item's values are not
	// overridden by the SDK's own defaults.
	ItemID string

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
	Fuel                       *Fuel

	ServiceCode     *string  // LC 116/2003 item.subitem, nfse only
	ServiceDiscount *float64 // unconditional discount, nfse only
	TaxRetained     bool     // ISSQN retained by the tomador, nfse only
	Observations    *string  // nfse only
}

func (p Product) ToDict() map[string]any {
	unit := p.Unit
	if unit == "" && p.ItemID == "" {
		unit = "UN"
	}
	quantity := p.Quantity
	if quantity == 0 {
		quantity = 1.0
	}

	data := map[string]any{
		"quantity":           quantity,
		"used_movable_asset": p.UsedMovableAsset,
	}
	if unit != "" {
		data["unit"] = unit
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
	if p.Fuel != nil {
		br["fuel"] = p.Fuel.toMap()
	}
	for k, v := range p.ExtraGroups {
		br[k] = v
	}

	if len(br) > 0 {
		data["br"] = br
	}

	result := map[string]any{"product": data}
	if p.ItemID != "" {
		result["item_id"] = p.ItemID
	}
	if p.ItemID == "" || p.Description != "" {
		result["description"] = p.Description
	}
	if p.ItemID == "" || p.TaxRetained {
		result["tax_retained"] = p.TaxRetained
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
