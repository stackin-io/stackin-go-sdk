// Freight, insurance, discount, and other expenses on the item.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:   "Produto com encargos adicionais",
		Amount:        200.00,
		NCM:           common.Ptr("94036000"),
		CFOP:          common.Ptr("5102"),
		Freight:       common.Ptr(15.00),
		Insurance:     common.Ptr(5.00),
		Discount:      common.Ptr(10.00),
		OtherExpenses: common.Ptr(3.50),
	}
	common.Issue(product, common.SameStateAddress)
}
