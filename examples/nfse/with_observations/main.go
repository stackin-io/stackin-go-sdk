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
		Description:  "Systems analysis and development",
		UnitPrice:    ptr(2400.00),
		ServiceCode:  ptr("1.01"),
		Observations: ptr("Referente ao contrato #2026-0042, etapa 2 de 3."),
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFSE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
