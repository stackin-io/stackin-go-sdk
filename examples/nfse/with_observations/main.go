// A free-text note attached to the service.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{
		Description:  "Systems analysis and development",
		Amount:       2400.00,
		ServiceCode:  common.Ptr("1.01"),
		Observations: common.Ptr("Referente ao contrato #2026-0042, etapa 2 de 3."),
	}
	common.Issue(product, nil)
}
