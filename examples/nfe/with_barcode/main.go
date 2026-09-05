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
		Description: "Produto com codigo de barras",
		Amount:      29.90,
		NCM:         ptr("21069090"),
		CFOP:        ptr("5102"),
		Barcode:     ptr("7891000100103"),
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
