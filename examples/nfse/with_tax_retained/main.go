// ISSQN retained by the tomador — rejected (E0583) if the issuer is MEI.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{
		Description: "UI/UX design",
		Amount:      3200.00,
		ServiceCode: common.Ptr("1.03"),
		TaxRetained: true,
	}
	common.Issue(product, &common.TomadorAddress)
}
