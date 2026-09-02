// Relevant-scale manufacturing indicator and its manufacturer CNPJ.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:      "Produto de fabricacao em escala",
		Amount:           150.00,
		NCM:              common.Ptr("87141000"),
		CFOP:             common.Ptr("5102"),
		CEST:             common.Ptr("0100100"),
		IndEscala:        common.Ptr("N"),
		ManufacturerCNPJ: common.Ptr("12345678000195"),
	}
	common.Issue(product, common.SameStateAddress)
}
