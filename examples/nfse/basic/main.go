// Only description/amount — service_code falls back to the company's fiscal profile.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{Description: "Software development", Amount: 5000.00}
	common.Issue(product, nil)
}
