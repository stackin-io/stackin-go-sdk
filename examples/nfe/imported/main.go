// An imported item, tracked by its Ficha de Conteudo de Importacao.
package main

import (
	"github.com/stackin-io/stackin-go-sdk/br"
	"github.com/stackin-io/stackin-go-sdk/examples/nfe/common"
)

func main() {
	product := br.Product{
		Description:                "Produto importado",
		Amount:                     320.00,
		NCM:                        common.Ptr("85171231"),
		CFOP:                       common.Ptr("5102"),
		ExTipi:                     common.Ptr("01"),
		ImportContentControlNumber: common.Ptr("550E8400-E29B-41D4-A716-446655440000"),
	}
	common.Issue(product, common.SameStateAddress)
}
