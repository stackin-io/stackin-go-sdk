// Linked to the buyer's purchase order and item number.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:       "Produto vinculado a pedido de compra",
		Amount:            75.00,
		NCM:               common.Ptr("84433210"),
		CFOP:              common.Ptr("5102"),
		PurchaseOrder:     common.Ptr("PC-2026-00042"),
		PurchaseOrderItem: common.Ptr("1"),
	}
	common.Issue(product, common.SameStateAddress)
}
