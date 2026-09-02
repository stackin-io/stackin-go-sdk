// An unconditional discount applied to the service value.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{
		Description:     "Monthly support and maintenance",
		Amount:          800.00,
		ServiceCode:     common.Ptr("1.07"),
		ServiceDiscount: common.Ptr(50.00),
	}
	common.Issue(product, nil)
}
