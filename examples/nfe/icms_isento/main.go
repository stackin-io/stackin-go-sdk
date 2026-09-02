// CSOSN 400 — MEI/Simples equivalent of the exempt ICMS40.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Rosa Holambra Vermelha",
		Amount:      112.44,
		NCM:         common.Ptr("06031100"),
		CFOP:        common.Ptr("6108"),
		Quantity:    6,
		Freight:     common.Ptr(11.05),
		Tax: &br.Tax{
			Icms:   br.IcmsSn102{Orig: common.Ptr("0"), CSOSN: "400"},
			Pis:    br.PisNt{CST: "07"},
			Cofins: br.CofinsNt{CST: "07"},
		},
	}
	common.Issue(product, common.OtherStateAddress)
}
