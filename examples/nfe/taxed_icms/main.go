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
		Description: "Plastico celofane 50x50",
		UnitPrice:   ptr(0.27),
		NCM:         ptr("39202019"),
		CFOP:        ptr("6108"),
		Freight:     ptr(0.03),
		Tax: &br.Tax{
			Icms: br.IcmsSn102{Orig: ptr("0"), CSOSN: "102"},
			Pis: br.PisAliq{
				CST: "01", VBC: "0.30", PPIS: "0.6500", VPIS: "0.00",
			},
			Cofins: br.CofinsAliq{
				CST: "01", VBC: "0.30", PCofins: "3.0000", VCofins: "0.01",
			},
		},
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
		RecipientAddress: &stackin.Address{
			Street:       "Avenida Atlantica",
			Number:       "500",
			Neighborhood: "Copacabana",
			City:         "Rio de Janeiro",
			State:        "RJ",
			ZipCode:      "22010000",
			CityCode:     "3304557",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
