// Multiple units at a per-unit price.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Caixa de parafusos",
		Amount:      12.50,
		NCM:         common.Ptr("73181500"),
		CFOP:        common.Ptr("5102"),
		Unit:        "CX",
		Quantity:    20,
	}
	common.Issue(product, common.SameStateAddress)
}
