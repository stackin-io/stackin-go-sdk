package main

import (
	"fmt"
	"log"
	"os"
	"time"

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
		Description:         "Technical consulting - 10 hours",
		UnitPrice:           ptr(1500.00),
		ServiceCode:         ptr("1.06"),
		NBS:                 "115021000",
		InternalCode:        "CONS01",
		ConditionalDiscount: ptr(10.00),
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFSE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
		Nfse: &br.NfseDetails{
			CompetenceDate:  time.Now(),
			ServiceCityCode: "3550308",
			ApproximateTaxes: &br.ApproximateTaxes{
				FederalPercent:   ptr(13.45),
				StatePercent:     ptr(0.0),
				MunicipalPercent: ptr(2.0),
			},
			References: &br.References{OrderNumber: "PO-1001"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result["warnings"])
}
