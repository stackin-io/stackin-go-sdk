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
		Description: "Rosa Holambra Vermelha",
		Amount:      112.44,
		NCM:         ptr("06031100"),
		CFOP:        ptr("6108"),
		Quantity:    6,
		Freight:     ptr(11.05),
		Tax: &br.Tax{
			Icms:   br.IcmsSn102{Orig: ptr("0"), CSOSN: "400"},
			Pis:    br.PisNt{CST: "07"},
			Cofins: br.CofinsNt{CST: "07"},
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
