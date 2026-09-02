// A used movable asset being resold.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:      "Bem movel usado",
		Amount:           500.00,
		NCM:              common.Ptr("87032310"),
		CFOP:             common.Ptr("5102"),
		UsedMovableAsset: true,
	}
	common.Issue(product, common.SameStateAddress)
}
