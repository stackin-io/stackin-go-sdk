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
	// RateState (pIBSUF), RateCity (pIBSMun) and RateFederal (pCBS) are in
	// percent and sent only when set, so 0 is a rate a pointer can say. An
	// NF-e group carries all three; an NFS-e group carries none.
	RateState   *float64
	RateCity    *float64
	RateFederal *float64
	// OperationIndicator is the cIndOp, 6 digits. Sent only when set.
	OperationIndicator *string
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
	}
	setIfNotNil(data, "rate_state", g.RateState)
	setIfNotNil(data, "rate_city", g.RateCity)
	setIfNotNil(data, "rate_federal", g.RateFederal)
	setIfNotNil(data, "operation_indicator", g.OperationIndicator)
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
	// TaxRetained is sent only when set (nil), so the item's values are not
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

	// NetWeight and GrossWeight are one unit's weight in kg, NF-e only. A
	// pointer so a real 0 can be sent.
	NetWeight   *float64
	GrossWeight *float64

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

	ServiceCode     *string // LC 116/2003 item.subitem, nfse only
	NBS             string
	ServiceDiscount *float64 // unconditional discount, nfse only
	// TaxRetained is a pointer so false can be sent on a line with an ItemID
	// (turning off a retention the item has). nil is left to the item, or
	// sent as false when there is no ItemID.
	TaxRetained  *bool   // ISSQN retained by the tomador, nfse only
	Observations *string // nfse only
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
	setIfNotNil(data, "net_weight", p.NetWeight)
	setIfNotNil(data, "gross_weight", p.GrossWeight)

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
	if p.TaxRetained != nil {
		result["tax_retained"] = *p.TaxRetained
	} else if p.ItemID == "" {
		result["tax_retained"] = false
	}
	if p.Amount != 0 {
		result["amount"] = decimalString(p.Amount)
	}
	if p.UnitPrice != nil {
		result["unit_price"] = decimalString(*p.UnitPrice)
	}
	setIfNotNil(result, "service_code", p.ServiceCode)
	if p.NBS != "" {
		result["nbs"] = p.NBS
	}
	setIfNotNil(result, "discount", p.ServiceDiscount)
	setIfNotNil(result, "observations", p.Observations)
	return result
}

// DecimalString writes a price as the plain decimal the API takes, never in
// exponent form. The catalog's unit price goes through it too.
func DecimalString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func decimalString(value float64) string {
	return DecimalString(value)
}

func setIfNotNil[T any](m map[string]any, key string, v *T) {
	if v != nil {
		m[key] = *v
	}
}
