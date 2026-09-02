// ICMS-ST item with a state tax benefit and presumed credit.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:    "Produto com beneficio fiscal",
		Amount:         80.00,
		NCM:            common.Ptr("22021000"),
		CFOP:           common.Ptr("5102"),
		CEST:           common.Ptr("0300700"),
		TaxBenefitCode: common.Ptr("PR820001"),
		PresumedCredits: []br.PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 2.40},
		},
	}
	common.Issue(product, common.SameStateAddress)
}
