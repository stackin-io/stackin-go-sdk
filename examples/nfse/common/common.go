package common

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func Ptr[T any](v T) *T {
	return &v
}

var TomadorAddress = stackin.Address{
	Street:       "Rua das Flores",
	Number:       "123",
	Neighborhood: "Centro",
	City:         "Sao Paulo",
	State:        "SP",
	ZipCode:      "01310100",
	CityCode:     "3550308",
}

func Issue(product br.Product, recipientAddress *stackin.Address) {
	godotenv.Load()
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType:     stackin.NFSE,
		ClientName:       "Comprador Teste Ltda",
		TaxID:            "11222333000181",
		Items:            []br.Product{product},
		RecipientAddress: recipientAddress,
	})

	switch e := err.(type) {
	case nil:
		fmt.Println("Issued:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
