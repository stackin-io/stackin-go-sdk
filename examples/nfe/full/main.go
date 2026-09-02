// Every field on Product filled in at once.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:      "Produto completo - todos os campos",
		Amount:           999.99,
		NCM:              common.Ptr("84713012"),
		CFOP:             common.Ptr("5102"),
		Unit:             "UN",
		Quantity:         2,
		Barcode:          common.Ptr("7891000100103"),
		CEST:             common.Ptr("0300700"),
		NVECodes:         []string{"NV0001", "NV0002"},
		IndEscala:        common.Ptr("N"),
		ManufacturerCNPJ: common.Ptr("12345678000195"),
		TaxBenefitCode:   common.Ptr("PR820001"),
		PresumedCredits: []br.PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 30.00},
		},
		ExTipi:                     common.Ptr("01"),
		Freight:                    common.Ptr(20.00),
		Insurance:                  common.Ptr(8.00),
		Discount:                   common.Ptr(15.00),
		OtherExpenses:              common.Ptr(5.00),
		UsedMovableAsset:           false,
		PurchaseOrder:              common.Ptr("PC-2026-00042"),
		PurchaseOrderItem:          common.Ptr("1"),
		ImportContentControlNumber: common.Ptr("550E8400-E29B-41D4-A716-446655440000"),
		RecopiNumber:               common.Ptr("00000000000012345678"),
		ExtraGroups:                map[string]any{},
	}
	common.Issue(product, common.SameStateAddress)
}
