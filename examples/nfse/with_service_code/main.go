// Explicit service_code, overriding the company default (1.06).
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfse/common"
)

func main() {
	product := br.Product{
		Description: "Technical consulting - 10 hours",
		Amount:      1500.00,
		ServiceCode: common.Ptr("1.06"),
	}
	common.Issue(product, nil)
}
