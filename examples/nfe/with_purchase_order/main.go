package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func ptr[T any](value T) *T {
	return &value
}

func main() {
	godotenv.Load()
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

	product := br.Product{
		Description:       "Produto vinculado a pedido de compra",
		UnitPrice:         ptr(75.00),
		NCM:               ptr("84433210"),
		CFOP:              ptr("5102"),
		PurchaseOrder:     ptr("PC-2026-00042"),
		PurchaseOrderItem: ptr("1"),
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
		RecipientAddress: &stackin.Address{
			Street:       "Rua das Palmeiras",
			Number:       "100",
			Neighborhood: "Centro",
			City:         "Florianopolis",
			State:        "SC",
			ZipCode:      "88010000",
			CityCode:     "4205407",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
