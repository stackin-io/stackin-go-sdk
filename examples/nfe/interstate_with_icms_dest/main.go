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
		Description: "Urso de Pelucia Dudu",
		UnitPrice:   ptr(92.72),
		NCM:         ptr("95030031"),
		CFOP:        ptr("6108"),
		Freight:     ptr(9.12),
		Tax: &br.Tax{
			Icms: br.IcmsSn900{
				Orig: ptr("0"), CSOSN: "900", ModBC: ptr("3"),
				VBC: ptr("101.84"), PICMS: ptr("12.0000"),
				VICMS: ptr("12.22"),
			},
			IcmsUfDest: &br.IcmsUfDest{
				VBCUFDest: "101.84", PICMSUFDest: "17.0000",
				PICMSInter: "12.00", PICMSInterPart: "100.0000",
				VICMSUFDest: "5.09", VICMSUFRemet: "0.00",
			},
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
