package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func ptr[T any](v T) *T {
	return &v
}

func basic() br.Product {
	return br.Product{Description: "Software development", Amount: 5000.00}
}

func withServiceCode() br.Product {
	return br.Product{
		Description: "Technical consulting - 10 hours",
		Amount:      1500.00,
		ServiceCode: ptr("1.06"),
	}
}

func withDiscount() br.Product {
	return br.Product{
		Description:     "Monthly support and maintenance",
		Amount:          800.00,
		ServiceCode:     ptr("1.07"),
		ServiceDiscount: ptr(50.00),
	}
}

func withTaxRetained() br.Product {
	return br.Product{
		Description: "UI/UX design",
		Amount:      3200.00,
		ServiceCode: ptr("1.03"),
		TaxRetained: true,
	}
}

func withObservations() br.Product {
	return br.Product{
		Description:  "Systems analysis and development",
		Amount:       2400.00,
		ServiceCode:  ptr("1.01"),
		Observations: ptr("Referente ao contrato #2026-0042, etapa 2 de 3."),
	}
}

func tomadorAddress() *stackin.Address {
	return &stackin.Address{
		Street:       "Rua das Flores",
		Number:       "123",
		Neighborhood: "Centro",
		City:         "Sao Paulo",
		State:        "SP",
		ZipCode:      "01310100",
		CityCode:     "3550308",
	}
}

func issue(client *stackin.Invoice, product br.Product, recipientAddress *stackin.Address) {
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

func main() {
	godotenv.Load()
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))

	issue(client, basic(), nil)
	issue(client, withServiceCode(), nil)
	issue(client, withDiscount(), nil)
	issue(client, withTaxRetained(), tomadorAddress())
	issue(client, withObservations(), nil)
}
