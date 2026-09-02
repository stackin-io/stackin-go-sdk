// Interstate sale, partilha do ICMS — CSOSN 900 (MEI/Simples).
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Urso de Pelucia Dudu",
		Amount:      92.72,
		NCM:         common.Ptr("95030031"),
		CFOP:        common.Ptr("6108"),
		Freight:     common.Ptr(9.12),
		Tax: &br.Tax{
			Icms: br.IcmsSn900{
				Orig: common.Ptr("0"), CSOSN: "900", ModBC: common.Ptr("3"),
				VBC: common.Ptr("101.84"), PICMS: common.Ptr("12.0000"),
				VICMS: common.Ptr("12.22"),
			},
			IcmsUfDest: &br.IcmsUfDest{
				VBCUFDest: "101.84", PICMSUFDest: "17.0000",
				PICMSInter: "12.00", PICMSInterPart: "100.0000",
				VICMSUFDest: "5.09", VICMSUFRemet: "0.00",
			},
			Pis:    br.PisNt{CST: "07"},
			Cofins: br.CofinsNt{CST: "07"},
		},
	}
	common.Issue(product, common.OtherStateAddress)
}
