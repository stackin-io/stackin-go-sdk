// CSOSN 102 (no credit) — MEI/Simples equivalent of ICMS00.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Plastico celofane 50x50",
		Amount:      0.27,
		NCM:         common.Ptr("39202019"),
		CFOP:        common.Ptr("6108"),
		Freight:     common.Ptr(0.03),
		Tax: &br.Tax{
			Icms: br.IcmsSn102{Orig: common.Ptr("0"), CSOSN: "102"},
			Pis: br.PisAliq{
				CST: "01", VBC: "0.30", PPIS: "0.6500", VPIS: "0.00",
			},
			Cofins: br.CofinsAliq{
				CST: "01", VBC: "0.30", PCofins: "3.0000", VCofins: "0.01",
			},
		},
	}
	common.Issue(product, common.OtherStateAddress)
}
