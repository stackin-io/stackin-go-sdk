package main

import (
	"fmt"
	"os"

	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func serviceCatalog() []br.Product {
	return []br.Product{
		{Description: "Software development", Amount: 5000.00},
		{Description: "Technical consulting - 10 hours", Amount: 1500.00},
		{Description: "Monthly support and maintenance", Amount: 800.00},
		{Description: "UI/UX design", Amount: 3200.00},
	}
}

func main() {
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFSE,
		ClientName:   "John Doe",
		TaxID:        "00000000000",
		Items:        serviceCatalog(),
	})

	switch e := err.(type) {
	case nil:
		fmt.Println("Issued:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach stackin-api")
	case *stackin.APIError:
		fmt.Printf("stackin-api rejected the request (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
