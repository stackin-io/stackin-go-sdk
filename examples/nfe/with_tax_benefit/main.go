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
		Description:    "Produto com beneficio fiscal",
		Amount:         80.00,
		NCM:            ptr("22021000"),
		CFOP:           ptr("5102"),
		CEST:           ptr("0300700"),
		TaxBenefitCode: ptr("PR820001"),
		PresumedCredits: []br.PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 2.40},
		},
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
