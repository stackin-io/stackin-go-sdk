package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./examples/reissue_invoice <invoice_id>")
		return
	}
	invoiceID := os.Args[1]

	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))
	result, err := client.Reissue(invoiceID)

	switch e := err.(type) {
	case nil:
		fmt.Println("Reissued:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
