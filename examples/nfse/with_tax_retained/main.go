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
		Description: "UI/UX design",
		UnitPrice:   ptr(3200.00),
		ServiceCode: ptr("1.03"),
		TaxRetained: true,
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFSE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
		RecipientAddress: &stackin.Address{
			Street:       "Rua das Flores",
			Number:       "123",
			Neighborhood: "Centro",
			City:         "Sao Paulo",
			State:        "SP",
			ZipCode:      "01310100",
			CityCode:     "3550308",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
