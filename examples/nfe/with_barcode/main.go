// A real GTIN/EAN instead of the "SEM GTIN" default.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Produto com codigo de barras",
		Amount:      29.90,
		NCM:         common.Ptr("21069090"),
		CFOP:        common.Ptr("5102"),
		Barcode:     common.Ptr("7891000100103"),
	}
	common.Issue(product, common.SameStateAddress)
}
