// Every optional field set at once.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{
		Description:     "Software licensing",
		Amount:          1200.00,
		ServiceCode:     common.Ptr("1.05"),
		ServiceDiscount: common.Ptr(100.00),
		TaxRetained:     true,
		Observations:    common.Ptr("Licenca anual, renovacao automatica."),
	}
	common.Issue(product, &common.TomadorAddress)
}
