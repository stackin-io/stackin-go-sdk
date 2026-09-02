// Only what NFE requires: description, amount, ncm, cfop.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description: "Produto basico",
		Amount:      50.00,
		NCM:         common.Ptr("84713012"),
		CFOP:        common.Ptr("5102"),
	}
	common.Issue(product, common.SameStateAddress)
}
